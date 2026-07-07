"use client";

import Link from 'next/link';
import * as React from 'react';
import { Bell, CheckCircle2 } from 'lucide-react';
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

export function NotificationDropdown() {
  const { isConnected } = useWebSocket();
  const {
    notifications,
    unreadCount,
    markNotificationsAsRead,
    actOnNotification,
  } = useNotifications();
  const [isOpen, setIsOpen] = React.useState(false);
  const [busyKey, setBusyKey] = React.useState<string | null>(null);
  const [error, setError] = React.useState<string | null>(null);

  const recentNotifications = notifications.slice(0, 6);

  React.useEffect(() => {
    if (!isOpen) return;

    const unreadVisibleIds = recentNotifications
      .filter((notification) => !notification.is_read)
      .map((notification) => notification.id);

    if (unreadVisibleIds.length === 0) return;

    void markNotificationsAsRead(unreadVisibleIds).catch(() => {
      setError('Could not update some notifications.');
    });
  }, [isOpen, markNotificationsAsRead, recentNotifications]);

  const handleDecision = async (
    event: React.MouseEvent<HTMLButtonElement>,
    notification: NotificationItem,
    decision: 'accept' | 'decline'
  ) => {
    event.stopPropagation();
    setBusyKey(`${notification.id}:${decision}`);
    setError(null);

    try {
      await actOnNotification(notification, decision);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not update this notification.');
    } finally {
      setBusyKey(null);
    }
  };

  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setIsOpen((prev) => !prev)}
        className="relative rounded-full p-2 transition-colors hover:bg-gray-100"
        aria-label="Open notifications"
      >
        <Bell className="w-5 h-5 text-gray-600" />
        {unreadCount > 0 && (
          <>
            <span className="absolute right-1.5 top-1.5 h-2.5 w-2.5 rounded-full border-2 border-white bg-red-500" />
            <span className="absolute -right-1 -top-1 min-w-5 rounded-full bg-red-500 px-1.5 py-0.5 text-[10px] font-semibold leading-none text-white">
              {unreadCount > 9 ? '9+' : unreadCount}
            </span>
          </>
        )}
      </button>

      {isOpen && (
        <div className="absolute right-0 z-50 mt-2 w-[22rem] overflow-hidden rounded-2xl border border-gray-100 bg-white shadow-lg">
          <div className="flex items-center justify-between border-b border-gray-100 bg-gray-50 px-4 py-3">
            <div>
              <h3 className="font-semibold text-gray-900">Notifications</h3>
              <p className="mt-0.5 text-xs text-gray-500">
                {unreadCount > 0 ? `${unreadCount} unread` : 'All caught up'}
              </p>
            </div>
            <span className="text-xs font-medium text-indigo-600">
              {isConnected ? 'Live' : 'Connecting...'}
            </span>
          </div>

          {error && (
            <div className="border-b border-red-100 bg-red-50 px-4 py-2 text-xs text-red-600">
              {error}
            </div>
          )}

          <div className="max-h-[28rem] overflow-y-auto">
            {recentNotifications.length === 0 ? (
              <div className="px-4 py-8 text-center text-sm text-gray-500">
                No notifications yet.
              </div>
            ) : (
              recentNotifications.map((notification) => {
                const link = getNotificationLink(notification);
                const actionable = isNotificationActionable(notification);

                return (
                  <div
                    key={notification.id}
                    className={`border-b border-gray-100 px-4 py-3 last:border-b-0 ${
                      notification.is_read ? 'bg-white' : 'bg-indigo-50/60'
                    }`}
                  >
                    <div className="flex items-start gap-3">
                      <div className="mt-0.5 rounded-full bg-white p-2 shadow-sm">
                        {getNotificationIcon(notification.type)}
                      </div>

                      <div className="min-w-0 flex-1">
                        <div className="flex items-start justify-between gap-2">
                          <div className="min-w-0">
                            <p className="text-sm font-semibold text-gray-900">
                              {getNotificationTitle(notification.type)}
                            </p>
                            <p className="mt-1 text-sm text-gray-600">
                              {getNotificationDescription(notification)}
                            </p>
                          </div>

                          {!notification.is_read && (
                            <span className="mt-1 h-2.5 w-2.5 shrink-0 rounded-full bg-indigo-500" />
                          )}
                        </div>

                        <div className="mt-2 flex items-center justify-between gap-2">
                          <p className="text-xs text-gray-400">
                            {formatNotificationTimestamp(notification.created_at)}
                          </p>
                          {notification.is_read && (
                            <span className="inline-flex items-center gap-1 text-xs text-emerald-600">
                              {notificationResolvedIcon}
                              Read
                            </span>
                          )}
                        </div>

                        {actionable && (
                          <div className="mt-3 flex flex-wrap gap-2">
                            {(['accept', 'decline'] as const).map((decision) => {
                              const actionKey = `${notification.id}:${decision}`;
                              const isBusy = busyKey === actionKey;
                              const isAccept = decision === 'accept';

                              return (
                                <button
                                  key={decision}
                                  type="button"
                                  onClick={(event) =>
                                    void handleDecision(event, notification, decision)
                                  }
                                  disabled={Boolean(busyKey)}
                                  className={`rounded-full px-3 py-1.5 text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-60 ${
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
                          </div>
                        )}

                        {!actionable && link && (
                          <div className="mt-3">
                            <Link
                              href={link}
                              onClick={() => setIsOpen(false)}
                              className="inline-flex items-center gap-2 text-xs font-medium text-indigo-600 hover:text-indigo-700"
                            >
                              <CheckCircle2 className="w-3.5 h-3.5" />
                              Open
                            </Link>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                );
              })
            )}
          </div>

          <div className="border-t border-gray-100 bg-gray-50 px-4 py-3">
            <Link
              href="/notifications"
              onClick={() => setIsOpen(false)}
              className="text-sm font-medium text-indigo-600 hover:text-indigo-700"
            >
              View all notifications
            </Link>
          </div>
        </div>
      )}
    </div>
  );
}
