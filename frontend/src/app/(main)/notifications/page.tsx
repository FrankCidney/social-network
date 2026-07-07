'use client';

import Link from 'next/link';
import * as React from 'react';
import { motion } from 'framer-motion';
import { Bell, CheckCircle2, Clock3 } from 'lucide-react';
import { useWebSocket } from '@/contexts/WebSocketContext';
import { useNotifications } from '@/contexts/NotificationsContext';
import {
  formatNotificationTimestamp,
  getNotificationActionLabel,
  getNotificationDescription,
  getNotificationIcon,
  getNotificationLink,
  getNotificationTitle,
  isNotificationActionable,
  notificationResolvedIcon,
} from '@/components/notifications/notificationUtils';
import type { NotificationItem } from '@/lib/api';

export default function NotificationsPage() {
  const { isConnected } = useWebSocket();
  const {
    notifications,
    loading,
    error,
    unreadCount,
    refresh,
    markAsRead,
    markAllAsRead,
    actOnNotification,
  } = useNotifications();
  const [markingAll, setMarkingAll] = React.useState(false);
  const [busyKey, setBusyKey] = React.useState<string | null>(null);
  const [actionError, setActionError] = React.useState<string | null>(null);
  const autoMarkedRef = React.useRef(false);

  React.useEffect(() => {
    if (loading || autoMarkedRef.current) return;

    autoMarkedRef.current = true;
    if (unreadCount === 0) return;

    void markAllAsRead().catch((err) => {
      setActionError(
        err instanceof Error ? err.message : 'Could not mark your notifications as read.'
      );
    });
  }, [loading, markAllAsRead, unreadCount]);

  const handleMarkAsRead = async (notificationId: string) => {
    setActionError(null);

    try {
      await markAsRead(notificationId);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Could not update this notification.');
    }
  };

  const handleMarkAllAsRead = async () => {
    if (unreadCount === 0) return;

    try {
      setMarkingAll(true);
      setActionError(null);
      await markAllAsRead();
    } catch (err) {
      setActionError(
        err instanceof Error ? err.message : 'Could not mark everything as read. Please try again.'
      );
    } finally {
      setMarkingAll(false);
    }
  };

  const handleDecision = async (
    notification: NotificationItem,
    decision: 'accept' | 'decline'
  ) => {
    setBusyKey(`${notification.id}:${decision}`);
    setActionError(null);

    try {
      await actOnNotification(notification, decision);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Could not update this notification.');
    } finally {
      setBusyKey(null);
    }
  };

  return (
    <div className="space-y-6">
      <div className="rounded-xl border border-gray-100 bg-white p-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="flex items-center gap-3 text-3xl font-bold text-gray-900">
              <div className="rounded-lg bg-amber-100 p-2">
                <Bell className="w-6 h-6 text-amber-600" />
              </div>
              Notifications
            </h1>
            <p className="mt-2 text-sm text-gray-500">
              Keep track of follow requests, group invites, approvals, and event updates in one place.
            </p>
          </div>

          <button
            onClick={handleMarkAllAsRead}
            disabled={markingAll || unreadCount === 0}
            className="inline-flex items-center gap-2 rounded-full border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <CheckCircle2 className="w-4 h-4" />
            {markingAll ? 'Marking…' : 'Mark all read'}
          </button>
        </div>

        <div className="mt-6 grid gap-4 sm:grid-cols-3">
          <div className="rounded-xl border border-gray-100 bg-gray-50 p-4">
            <p className="text-sm text-gray-500">Unread</p>
            <p className="mt-2 text-2xl font-semibold text-gray-900">{unreadCount}</p>
          </div>
          <div className="rounded-xl border border-gray-100 bg-gray-50 p-4">
            <p className="text-sm text-gray-500">Live updates</p>
            <p className="mt-2 text-2xl font-semibold text-gray-900">{isConnected ? 'On' : 'Connecting'}</p>
          </div>
          <div className="rounded-xl border border-gray-100 bg-gray-50 p-4">
            <p className="text-sm text-gray-500">Total</p>
            <p className="mt-2 text-2xl font-semibold text-gray-900">{notifications.length}</p>
          </div>
        </div>
      </div>

      <div className="space-y-4">
        {loading && (
          <div className="space-y-4">
            {[1, 2, 3].map((i) => (
              <div key={i} className="animate-pulse rounded-xl border border-gray-100 bg-white p-4">
                <div className="flex items-start gap-3">
                  <div className="h-9 w-9 shrink-0 rounded-full bg-gray-100" />
                  <div className="flex-1 space-y-2">
                    <div className="h-3 w-40 rounded bg-gray-100" />
                    <div className="h-3 w-64 rounded bg-gray-100" />
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {!loading && actionError && (
          <div className="rounded-xl border border-red-100 bg-red-50 px-4 py-3 text-sm text-red-600">
            {actionError}
          </div>
        )}

        {!loading && error && (
          <div className="rounded-xl border border-gray-100 bg-white p-8 text-center">
            <p className="mb-3 text-sm text-red-600">{error}</p>
            <button onClick={refresh} className="text-sm text-indigo-600 hover:underline">
              Retry
            </button>
          </div>
        )}

        {!loading && !error && notifications.length === 0 && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            className="rounded-xl border border-gray-100 bg-white p-12 text-center"
          >
            <Bell className="mx-auto mb-4 h-12 w-12 text-gray-300" />
            <h3 className="mb-2 text-lg font-bold text-gray-900">No notifications yet</h3>
            <p className="text-sm text-gray-500">
              New activity will appear here as soon as the app sends it.
            </p>
          </motion.div>
        )}

        {!loading &&
          !error &&
          notifications.map((notification, index) => {
            const actionable = isNotificationActionable(notification);
            const link = getNotificationLink(notification);

            return (
              <motion.div
                key={notification.id}
                initial={{ opacity: 0, y: 12 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: index * 0.04 }}
                onClick={() => {
                  if (!notification.is_read) {
                    void handleMarkAsRead(notification.id);
                  }
                }}
                className={`rounded-xl border p-4 transition-colors ${
                  notification.is_read
                    ? 'border-gray-100 bg-white'
                    : 'cursor-pointer border-indigo-100 bg-indigo-50/70 hover:bg-indigo-50'
                }`}
              >
                <div className="flex items-start gap-3">
                  <div className="mt-0.5 rounded-full bg-white p-2 shadow-sm">
                    {getNotificationIcon(notification.type)}
                  </div>
                  <div className="flex-1">
                    <div className="flex flex-wrap items-start justify-between gap-2">
                      <div>
                        <div className="flex items-center gap-2">
                          <p className="font-semibold text-gray-900">
                            {getNotificationTitle(notification.type)}
                          </p>
                          {!notification.is_read && (
                            <span className="h-2.5 w-2.5 rounded-full bg-indigo-500" />
                          )}
                        </div>
                        <p className="mt-1 text-sm text-gray-600">
                          {getNotificationDescription(notification)}
                        </p>
                      </div>
                      <div className="flex items-center gap-2 text-xs text-gray-500">
                        <Clock3 className="w-3.5 h-3.5" />
                        {formatNotificationTimestamp(notification.created_at)}
                      </div>
                    </div>

                    <div className="mt-4 flex flex-wrap items-center gap-2">
                      {actionable &&
                        (['accept', 'decline'] as const).map((decision) => {
                          const actionKey = `${notification.id}:${decision}`;
                          const isBusy = busyKey === actionKey;
                          const isAccept = decision === 'accept';

                          return (
                            <button
                              key={decision}
                              type="button"
                              onClick={(event) => {
                                event.stopPropagation();
                                void handleDecision(notification, decision);
                              }}
                              disabled={Boolean(busyKey)}
                              className={`rounded-full px-4 py-2 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-60 ${
                                isAccept
                                  ? 'bg-indigo-600 text-white hover:bg-indigo-700'
                                  : 'border border-gray-200 text-gray-700 hover:bg-gray-50'
                              }`}
                            >
                              {isBusy
                                ? 'Working...'
                                : getNotificationActionLabel(notification, isAccept)}
                            </button>
                          );
                        })}

                      {link && (
                        <Link
                          href={link}
                          onClick={(event) => {
                            event.stopPropagation();
                          }}
                          className="text-sm font-medium text-indigo-600 hover:text-indigo-700"
                        >
                          Open related page
                        </Link>
                      )}

                      {notification.is_read && (
                        <span className="inline-flex items-center gap-1 text-sm text-emerald-600">
                          {notificationResolvedIcon}
                          Read
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              </motion.div>
            );
          })}
      </div>
    </div>
  );
}
