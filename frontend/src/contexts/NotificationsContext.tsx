"use client";

import * as React from 'react';
import {
  groupAPI,
  isAuthenticationError,
  NotificationItem,
  NotificationListResponse,
  notificationsAPI,
} from '@/lib/api';
import { useWebSocket } from '@/contexts/WebSocketContext';
import { getNotificationDescription } from '@/components/notifications/notificationUtils';

type NotificationDecision = 'accept' | 'decline';

type NotificationsContextValue = {
  notifications: NotificationItem[];
  unreadCount: number;
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
  markAsRead: (notificationId: string) => Promise<void>;
  markAllAsRead: () => Promise<void>;
  markNotificationsAsRead: (notificationIds: string[]) => Promise<void>;
  actOnNotification: (
    notification: NotificationItem,
    decision: NotificationDecision
  ) => Promise<void>;
};

const NotificationsContext = React.createContext<NotificationsContextValue | undefined>(
  undefined
);

function unwrapNotifications(
  data: NotificationListResponse | NotificationItem[] | null | undefined
): NotificationItem[] {
  if (!data) {
    return [];
  }

  if (Array.isArray(data)) {
    return data;
  }

  return Array.isArray(data.notifications) ? data.notifications : [];
}

function mergeNotification(
  current: NotificationItem[],
  incoming: NotificationItem
): NotificationItem[] {
  const next = {
    ...incoming,
    message: incoming.message ?? getNotificationDescription(incoming),
  };
  const existingIndex = current.findIndex((item) => item.id === next.id);

  if (existingIndex === -1) {
    return [next, ...current].sort(
      (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    );
  }

  const updated = [...current];
  updated[existingIndex] = {
    ...updated[existingIndex],
    ...next,
  };
  return updated;
}

export function NotificationsProvider({ children }: { children: React.ReactNode }) {
  const { socket } = useWebSocket();
  const [notifications, setNotifications] = React.useState<NotificationItem[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const notificationsRef = React.useRef<NotificationItem[]>([]);
  const [resolvedNotificationIds, setResolvedNotificationIds] = React.useState<string[]>([]);
  const resolvedNotificationIdsRef = React.useRef<string[]>([]);

  React.useEffect(() => {
    try {
      const stored = window.localStorage.getItem('resolved-notification-ids');
      if (!stored) return;

      const parsed = JSON.parse(stored);
      if (Array.isArray(parsed)) {
        setResolvedNotificationIds(
          parsed.filter((value): value is string => typeof value === 'string')
        );
      }
    } catch {
      // Ignore invalid local data and rely on server state.
    }
  }, []);

  React.useEffect(() => {
    notificationsRef.current = notifications;
  }, [notifications]);

  React.useEffect(() => {
    resolvedNotificationIdsRef.current = resolvedNotificationIds;
    try {
      window.localStorage.setItem(
        'resolved-notification-ids',
        JSON.stringify(resolvedNotificationIds)
      );
    } catch {
      // Local persistence is best effort only.
    }
  }, [resolvedNotificationIds]);

  const decorateNotification = React.useCallback(
    (item: NotificationItem): NotificationItem => ({
      ...item,
      is_resolved:
        Boolean(item.is_resolved) || resolvedNotificationIdsRef.current.includes(item.id),
      message: item.message ?? getNotificationDescription(item),
    }),
    []
  );

  const refresh = React.useCallback(async () => {
    try {
      setLoading(true);
      const data = await notificationsAPI.getNotifications();
      setNotifications(
        unwrapNotifications(data).map((item) => decorateNotification(item))
      );
      setError(null);
    } catch (err) {
      if (!isAuthenticationError(err)) {
        setError(err instanceof Error ? err.message : 'Could not load your notifications.');
      }
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    void refresh();
  }, [refresh]);

  React.useEffect(() => {
    if (!socket) return;

    const handleMessage = (event: MessageEvent) => {
      try {
        const data = JSON.parse(event.data);
        if (
          data.type === 'notification' ||
          data.type === 'follow_request' ||
          data.type === 'follow_accepted' ||
          data.type === 'group_invite' ||
          data.type === 'group_join_request' ||
          data.type === 'group_event'
        ) {
          const payload = data.payload as NotificationItem;
          setNotifications((prev) => mergeNotification(prev, decorateNotification(payload)));
        }
      } catch {
        // Ignore malformed websocket payloads; the page should keep working.
      }
    };

    socket.addEventListener('message', handleMessage);
    return () => socket.removeEventListener('message', handleMessage);
  }, [decorateNotification, socket]);

  const markResolved = React.useCallback((notificationId: string) => {
    setResolvedNotificationIds((prev) => {
      if (prev.includes(notificationId)) return prev;
      return [...prev, notificationId];
    });
    setNotifications((prev) =>
      prev.map((item) =>
        item.id === notificationId ? { ...item, is_resolved: true, is_read: true } : item
      )
    );
  }, []);

  const markNotificationsAsRead = React.useCallback(async (notificationIds: string[]) => {
    const ids = Array.from(new Set(notificationIds)).filter(Boolean);
    if (ids.length === 0) return;

    const previous = notificationsRef.current;
    const idSet = new Set(ids);

    setNotifications((prev) =>
      prev.map((item) => (idSet.has(item.id) ? { ...item, is_read: true } : item))
    );

    const results = await Promise.allSettled(ids.map((id) => notificationsAPI.markAsRead(id)));
    const failedIds = ids.filter((_, index) => results[index].status === 'rejected');

    if (failedIds.length > 0) {
      const failedSet = new Set(failedIds);
      setNotifications((prev) =>
        prev.map((item) => {
          if (!failedSet.has(item.id)) return item;
          const prior = previous.find((candidate) => candidate.id === item.id);
          return prior ? { ...item, is_read: prior.is_read } : item;
        })
      );

      const firstFailure = results.find(
        (result): result is PromiseRejectedResult => result.status === 'rejected'
      );
      throw firstFailure?.reason ?? new Error('Could not update some notifications.');
    }
  }, []);

  const markAsRead = React.useCallback(
    async (notificationId: string) => {
      await markNotificationsAsRead([notificationId]);
    },
    [markNotificationsAsRead]
  );

  const markAllAsRead = React.useCallback(async () => {
    const previous = notificationsRef.current;
    const unreadIds = previous.filter((item) => !item.is_read).map((item) => item.id);

    if (unreadIds.length === 0) return;

    setNotifications((prev) => prev.map((item) => ({ ...item, is_read: true })));

    try {
      await notificationsAPI.markAllAsRead();
    } catch (err) {
      setNotifications(previous);
      throw err;
    }
  }, []);

  const actOnNotification = React.useCallback(
    async (notification: NotificationItem, decision: NotificationDecision) => {
      const accept = decision === 'accept';

      switch (notification.type) {
        case 'follow_request':
          await notificationsAPI.respondToFollowRequest(notification.actor_id, accept);
          break;
        case 'group_invite':
          if (!notification.group_id) {
            throw new Error('This invite is missing its group reference.');
          }
          if (accept) {
            await groupAPI.acceptInvite(notification.group_id);
          } else {
            await groupAPI.declineInvite(notification.group_id);
          }
          break;
        case 'group_join_request':
          if (!notification.group_id) {
            throw new Error('This join request is missing its group reference.');
          }
          if (accept) {
            await groupAPI.acceptJoinRequest(notification.group_id, notification.actor_id);
          } else {
            await groupAPI.declineJoinRequest(notification.group_id, notification.actor_id);
          }
          break;
        default:
          throw new Error('This notification does not support actions.');
      }

      markResolved(notification.id);
      await notificationsAPI.resolve(notification.id);
    },
    [markResolved]
  );

  const value = React.useMemo(
    () => ({
      notifications,
      unreadCount: notifications.filter((item) => !item.is_read).length,
      loading,
      error,
      refresh,
      markAsRead,
      markAllAsRead,
      markNotificationsAsRead,
      actOnNotification,
    }),
    [
      actOnNotification,
      error,
      loading,
      markAllAsRead,
      markAsRead,
      markNotificationsAsRead,
      notifications,
      refresh,
    ]
  );

  return (
    <NotificationsContext.Provider value={value}>
      {children}
    </NotificationsContext.Provider>
  );
}

export function useNotifications() {
  const context = React.useContext(NotificationsContext);
  if (!context) {
    throw new Error('useNotifications must be used within a NotificationsProvider');
  }
  return context;
}
