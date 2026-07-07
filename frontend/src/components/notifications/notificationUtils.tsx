import { Bell, CheckCircle2, MessageSquare, Sparkles, Users } from 'lucide-react';
import type { NotificationItem } from '@/lib/api';

export function formatNotificationTimestamp(value: string) {
  const date = new Date(value);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);

  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins} minute${diffMins === 1 ? '' : 's'} ago`;

  const diffHours = Math.floor(diffMins / 60);
  if (diffHours < 24) return `${diffHours} hour${diffHours === 1 ? '' : 's'} ago`;

  const diffDays = Math.floor(diffHours / 24);
  return `${diffDays} day${diffDays === 1 ? '' : 's'} ago`;
}

export function getNotificationActorLabel(notification: NotificationItem) {
  const actor = notification.actor;

  if (actor) {
    const fullName = `${actor.first_name} ${actor.last_name}`.trim();
    if (fullName) return fullName;
    if (actor.nickname) return actor.nickname;
  }

  return notification.actor_id ? `User ${notification.actor_id}` : 'Someone';
}

export function getNotificationTitle(type: string) {
  switch (type) {
    case 'follow_request':
      return 'Follow request';
    case 'follow_accepted':
      return 'Follow accepted';
    case 'group_invite':
      return 'Group invite';
    case 'group_join_request':
      return 'Group join request';
    case 'group_event':
      return 'New group event';
    case 'notification':
      return 'New activity';
    default:
      return 'Notification';
  }
}

export function getNotificationDescription(notification: NotificationItem) {
  if (notification.message) {
    return notification.message;
  }

  const actorLabel = getNotificationActorLabel(notification);

  switch (notification.type) {
    case 'follow_request':
      return `${actorLabel} wants to follow you.`;
    case 'follow_accepted':
      return `${actorLabel} accepted your follow request.`;
    case 'group_invite':
      return `${actorLabel} invited you to join a group.`;
    case 'group_join_request':
      return `${actorLabel} requested to join your group.`;
    case 'group_event':
      return `${actorLabel} created a new group event.`;
    case 'notification':
      return `${actorLabel} sent you an update.`;
    default:
      return `${actorLabel} sent an update.`;
  }
}

export function getNotificationIcon(type: string) {
  switch (type) {
    case 'follow_request':
    case 'follow_accepted':
      return <Users className="w-5 h-5 text-indigo-600" />;
    case 'group_invite':
    case 'group_join_request':
    case 'group_event':
      return <Sparkles className="w-5 h-5 text-amber-600" />;
    case 'notification':
      return <MessageSquare className="w-5 h-5 text-emerald-600" />;
    default:
      return <Bell className="w-5 h-5 text-slate-600" />;
  }
}

export function getNotificationLink(notification: NotificationItem) {
  switch (notification.type) {
    case 'group_invite':
    case 'group_join_request':
    case 'group_event':
      return notification.group_id ? `/groups/${notification.group_id}` : null;
    case 'follow_request':
    case 'follow_accepted':
      return notification.actor_id ? `/profile/${notification.actor_id}` : null;
    default:
      return null;
  }
}

export function isNotificationActionable(notification: NotificationItem) {
  if (notification.is_resolved) {
    return false;
  }

  return (
    notification.type === 'follow_request' ||
    notification.type === 'group_invite' ||
    notification.type === 'group_join_request'
  );
}

export function getNotificationActionLabel(
  notification: NotificationItem,
  accept: boolean
) {
  switch (notification.type) {
    case 'follow_request':
      return accept ? 'Accept' : 'Decline';
    case 'group_invite':
      return accept ? 'Accept invite' : 'Decline invite';
    case 'group_join_request':
      return accept ? 'Approve' : 'Decline';
    default:
      return accept ? 'Accept' : 'Decline';
  }
}

export const notificationResolvedIcon = (
  <CheckCircle2 className="w-4 h-4 text-emerald-600" />
);
