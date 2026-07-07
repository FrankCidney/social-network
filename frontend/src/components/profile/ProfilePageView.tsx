'use client';

import * as React from 'react';
import Link from 'next/link';
import {
  BookUser,
  Calendar,
  Camera,
  CheckCircle2,
  Globe2,
  Lock,
  Mail,
  MoreHorizontal,
  Pencil,
  UserMinus,
  UserPlus,
  Users,
  X,
} from 'lucide-react';
import { Button } from '@/components/ui/Button';
import {
  feedAPI,
  followAPI,
  isAuthenticationError,
  isForbiddenError,
  PostResponse,
  profileAPI,
  ProfileResponse,
  PublicUser,
  resolveAssetUrl,
  UpdateProfilePayload,
} from '@/lib/api';

const LIST_LIMIT = 20;

function getDisplayName(user: PublicUser) {
  return user.nickname?.trim() ? user.nickname : `${user.first_name} ${user.last_name}`;
}

function getInitials(user: PublicUser) {
  return `${user.first_name?.[0] ?? ''}${user.last_name?.[0] ?? ''}`.toUpperCase();
}

function formatDate(value?: string) {
  if (!value) return 'Not provided';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return 'Not provided';
  return date.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
}

function formatShortDate(value?: string) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
}

function toDateInputValue(value?: string) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  return date.toISOString().slice(0, 10);
}

function formatPostPrivacyLabel(privacy: string) {
  switch (privacy) {
    case 'almost_private':
      return 'almost private';
    default:
      return privacy;
  }
}

type EditFormState = {
  first_name: string;
  last_name: string;
  nickname: string;
  about_me: string;
  dob: string;
  is_public: boolean;
};

type StatsPanel = 'followers' | 'following' | 'posts';

type StatsPanelState = {
  type: StatsPanel;
  loading: boolean;
  error: string | null;
  users: PublicUser[];
  posts: PostResponse[];
};

function emptyPanel(type: StatsPanel): StatsPanelState {
  return {
    type,
    loading: true,
    error: null,
    users: [],
    posts: [],
  };
}

export function ProfilePageView({ userId }: { userId?: string }) {
  const [profile, setProfile] = React.useState<ProfileResponse | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  const [isEditing, setIsEditing] = React.useState(false);
  const [form, setForm] = React.useState<EditFormState | null>(null);
  const [saving, setSaving] = React.useState(false);
  const [saveError, setSaveError] = React.useState<string | null>(null);
  const [followError, setFollowError] = React.useState<string | null>(null);

  const [avatarFile, setAvatarFile] = React.useState<File | null>(null);
  const [avatarPreview, setAvatarPreview] = React.useState<string | null>(null);
  const [uploadingAvatar, setUploadingAvatar] = React.useState(false);
  const [followLoading, setFollowLoading] = React.useState(false);
  const [panel, setPanel] = React.useState<StatsPanelState | null>(null);
  const avatarInputRef = React.useRef<HTMLInputElement>(null);

  const loadProfile = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = userId ? await profileAPI.getProfile(userId) : await profileAPI.getMyProfile();
      setProfile(data);
    } catch (err) {
      if (!isAuthenticationError(err)) {
        setError(
          err instanceof Error
            ? err.message
            : userId
              ? 'Unable to load this profile right now.'
              : 'Unable to load your profile right now.'
        );
      }
    } finally {
      setLoading(false);
    }
  }, [userId]);

  React.useEffect(() => {
    void loadProfile();
  }, [loadProfile]);

  const isOwnProfile = profile?.is_own_profile ?? !userId;
  const isFollowing = Boolean(profile?.is_following);
  const canViewFullProfile =
    profile?.can_view_full_profile ?? (profile ? isOwnProfile || profile.user.is_public || isFollowing : false);

  React.useEffect(() => {
    return () => {
      if (avatarPreview) URL.revokeObjectURL(avatarPreview);
    };
  }, [avatarPreview]);

  const startEditing = () => {
    if (!profile) return;
    setFollowError(null);
    setForm({
      first_name: profile.user.first_name,
      last_name: profile.user.last_name,
      nickname: profile.user.nickname ?? '',
      about_me: profile.about_me ?? '',
      dob: toDateInputValue(profile.dob),
      is_public: profile.user.is_public,
    });
    setSaveError(null);
    setIsEditing(true);
  };

  const cancelEditing = () => {
    setIsEditing(false);
    setForm(null);
    setSaveError(null);
    setAvatarFile(null);
    setAvatarPreview(null);
    if (avatarInputRef.current) avatarInputRef.current.value = '';
  };

  const handleAvatarSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (avatarPreview) URL.revokeObjectURL(avatarPreview);
    setAvatarFile(file);
    setAvatarPreview(URL.createObjectURL(file));
  };

  const handleSave = async () => {
    if (!form || !profile) return;
    setSaving(true);
    setSaveError(null);
    setFollowError(null);

    try {
      if (avatarFile) {
        setUploadingAvatar(true);
        await profileAPI.uploadAvatar(avatarFile);
        setUploadingAvatar(false);
      }

      const payload: UpdateProfilePayload = {
        first_name: form.first_name.trim(),
        last_name: form.last_name.trim(),
        nickname: form.nickname.trim() || undefined,
        about_me: form.about_me.trim(),
        dob: form.dob || undefined,
        is_public: form.is_public,
      };

      const updated = await profileAPI.updateProfile(payload);
      setProfile(updated);
      setIsEditing(false);
      setForm(null);
      setAvatarFile(null);
      setAvatarPreview(null);
      if (avatarInputRef.current) avatarInputRef.current.value = '';
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Could not save your changes. Please try again.');
    } finally {
      setSaving(false);
      setUploadingAvatar(false);
    }
  };

  const handleFollowToggle = async () => {
    if (!profile || profile.is_own_profile !== false) return;

    const wasFollowing = Boolean(profile.is_following);
    setFollowLoading(true);
    setFollowError(null);

    try {
      if (wasFollowing) {
        await followAPI.unfollow(profile.user.id);
      } else {
        await followAPI.follow(profile.user.id);
      }

      setProfile((current) => {
        if (!current) return current;
        const createsPendingRequest = !wasFollowing && !current.user.is_public;

        return {
          ...current,
          follower_count: createsPendingRequest
            ? current.follower_count
            : Math.max(0, current.follower_count + (wasFollowing ? -1 : 1)),
          is_following: createsPendingRequest ? false : !wasFollowing,
          follow_request_status: createsPendingRequest ? 'pending' : undefined,
        };
      });
    } catch (err) {
      setFollowError(err instanceof Error ? err.message : 'Could not update follow state.');
    } finally {
      setFollowLoading(false);
    }
  };

  const openStatsPanel = async (type: StatsPanel) => {
    if (!profile) return;
    if (!canViewFullProfile) {
      return;
    }
    setPanel(emptyPanel(type));

    try {
      if (type === 'followers') {
        const data = await profileAPI.getFollowers(profile.user.id, LIST_LIMIT, 0);
        setPanel({ type, loading: false, error: null, users: data.users, posts: [] });
        return;
      }

      if (type === 'following') {
        const data = await profileAPI.getFollowing(profile.user.id, LIST_LIMIT, 0);
        setPanel({ type, loading: false, error: null, users: data.users, posts: [] });
        return;
      }

      const data = await feedAPI.getUserPosts(profile.user.id, LIST_LIMIT, 0);
      setPanel({ type, loading: false, error: null, users: [], posts: data.posts });
    } catch (err) {
      setPanel({
        type,
        loading: false,
        error: isForbiddenError(err)
          ? 'This content is private.'
          : err instanceof Error
            ? err.message
            : 'Could not load this list.',
        users: [],
        posts: [],
      });
    }
  };

  if (loading) {
    return (
      <div className="space-y-4">
        <div className="animate-pulse rounded-xl border border-gray-100 bg-white p-6">
          <div className="h-8 w-64 rounded bg-gray-100 mb-4" />
          <div className="grid gap-4 sm:grid-cols-3">
            <div className="h-24 rounded-xl bg-gray-100" />
            <div className="h-24 rounded-xl bg-gray-100" />
            <div className="h-24 rounded-xl bg-gray-100" />
          </div>
        </div>
        <div className="animate-pulse rounded-xl border border-gray-100 bg-white p-6 h-72" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-xl p-6 text-red-700">
        <h1 className="text-xl font-semibold">Profile unavailable</h1>
        <p className="mt-2 text-sm">{error}</p>
        <button onClick={loadProfile} className="mt-3 text-sm font-medium underline">
          Try again
        </button>
      </div>
    );
  }

  if (!profile) {
    return null;
  }

  const isPrivateLimited = !canViewFullProfile;
  const avatarUrl = avatarPreview || resolveAssetUrl(profile.user.avatar_path);
  const name = getDisplayName(profile.user);
  const handle = `@${profile.user.first_name.toLowerCase()}${profile.user.last_name.toLowerCase()}`;
  const email = isOwnProfile ? profile.email ?? profile.user.email : undefined;

  if (isEditing && form) {
    return (
      <div className="space-y-6">
        <div className="rounded-xl border border-gray-100 bg-white p-6">
          <div className="flex flex-col gap-6 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">Edit profile</h1>
              <p className="mt-1 text-sm text-gray-500">
                Update the information people see on your profile.
              </p>
            </div>

            <div className="flex items-center gap-2">
              <Button
                variant="ghost"
                size="lg"
                onClick={cancelEditing}
                disabled={saving}
                className="flex items-center gap-2"
              >
                <X className="w-4 h-4" />
                Cancel
              </Button>
              <Button
                variant="primary"
                size="lg"
                onClick={handleSave}
                disabled={saving || !form.first_name.trim() || !form.last_name.trim()}
                className="flex items-center gap-2"
              >
                {saving ? (uploadingAvatar ? 'Uploading photo…' : 'Saving…') : 'Save changes'}
              </Button>
            </div>
          </div>

          {saveError && <p className="mt-4 text-sm text-red-600">{saveError}</p>}

          <div className="mt-8 grid gap-8 lg:grid-cols-[220px_1fr]">
            <div className="space-y-4">
              <div className="relative mx-auto flex h-32 w-32 items-center justify-center overflow-hidden rounded-xl bg-indigo-100 text-4xl font-bold text-indigo-700">
                {avatarUrl ? (
                  <img src={avatarUrl} alt={name} className="h-full w-full object-cover" />
                ) : (
                  getInitials(profile.user)
                )}
              </div>
              <Button
                variant="secondary"
                size="md"
                onClick={() => avatarInputRef.current?.click()}
                className="mx-auto flex items-center gap-2"
                disabled={saving}
              >
                <Camera className="h-4 w-4" />
                Change photo
              </Button>
              <input
                ref={avatarInputRef}
                type="file"
                accept="image/*"
                className="hidden"
                onChange={handleAvatarSelect}
              />
            </div>

            <div className="grid gap-5">
              <div className="grid gap-4 sm:grid-cols-2">
                <LabeledField label="First name">
                  <input
                    value={form.first_name}
                    onChange={(e) => setForm({ ...form, first_name: e.target.value })}
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  />
                </LabeledField>
                <LabeledField label="Last name">
                  <input
                    value={form.last_name}
                    onChange={(e) => setForm({ ...form, last_name: e.target.value })}
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  />
                </LabeledField>
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                <LabeledField label="Nickname">
                  <input
                    value={form.nickname}
                    onChange={(e) => setForm({ ...form, nickname: e.target.value })}
                    placeholder="Optional"
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  />
                </LabeledField>
                <LabeledField label="Birthday">
                  <input
                    type="date"
                    value={form.dob}
                    onChange={(e) => setForm({ ...form, dob: e.target.value })}
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  />
                </LabeledField>
              </div>

              <LabeledField label="About me">
                <textarea
                  value={form.about_me}
                  onChange={(e) => setForm({ ...form, about_me: e.target.value })}
                  rows={5}
                  placeholder="Tell people a bit about yourself..."
                  className="w-full resize-none rounded-xl border border-gray-200 px-4 py-3 text-sm text-gray-700 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
              </LabeledField>

              <label className="flex items-start justify-between gap-4 rounded-xl border border-gray-100 bg-gray-50 p-4">
                <span>
                  <span className="block text-sm font-semibold text-gray-900">Public profile</span>
                  <span className="mt-1 block text-sm text-gray-500">
                    Public profiles can be viewed by anyone. Private profiles show full details only to followers.
                  </span>
                </span>
                <input
                  type="checkbox"
                  checked={form.is_public}
                  onChange={(e) => setForm({ ...form, is_public: e.target.checked })}
                  className="mt-1 h-5 w-5 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
                />
              </label>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="bg-white rounded-xl border border-gray-100 p-6">
        <div className="flex flex-col gap-6 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex items-center gap-5">
            <div className="flex h-20 w-20 items-center justify-center overflow-hidden rounded-xl bg-indigo-100 text-3xl font-bold text-indigo-700">
              {avatarUrl ? (
                <img src={avatarUrl} alt={name} className="h-full w-full object-cover" />
              ) : (
                getInitials(profile.user)
              )}
            </div>
            <div>
              <h1 className="text-3xl font-bold text-gray-900">{name}</h1>
              <div className="mt-1 flex flex-wrap items-center gap-3 text-sm text-gray-500">
                <span>{handle}</span>
                {email && (
                  <span className="inline-flex items-center gap-1">
                    <Mail className="h-4 w-4" />
                    {email}
                  </span>
                )}
              </div>

              <div className="mt-3 inline-flex items-center gap-2 rounded-full border border-gray-200 bg-gray-50 px-3 py-1 text-sm text-gray-600">
                {profile.user.is_public ? (
                  <Globe2 className="w-4 h-4 text-emerald-500" />
                ) : (
                  <Lock className="w-4 h-4 text-amber-500" />
                )}
                {profile.user.is_public ? 'Public profile' : 'Private profile'}
              </div>
            </div>
          </div>

          {isOwnProfile ? (
            <Button variant="secondary" size="lg" onClick={startEditing} className="flex items-center gap-2">
              <Pencil className="w-4 h-4" />
              Edit profile
            </Button>
          ) : (
            <div className="flex flex-wrap items-center gap-3">
              <Button
                variant={isFollowing ? 'secondary' : 'primary'}
                size="lg"
                onClick={handleFollowToggle}
                disabled={followLoading || profile.follow_request_status === 'pending'}
                className="flex items-center gap-2"
              >
                {isFollowing ? <UserMinus className="w-4 h-4" /> : <UserPlus className="w-4 h-4" />}
                {profile.follow_request_status === 'pending'
                  ? 'Requested'
                  : isFollowing
                    ? 'Unfollow'
                    : 'Follow'}
              </Button>
              <Link
                href={`/messages?user=${encodeURIComponent(profile.user.id)}`}
                className="inline-flex items-center justify-center rounded-bento border border-gray-200 bg-white px-6 py-3 text-lg font-medium text-text-main transition-colors hover:bg-gray-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
              >
                Message
              </Link>
            </div>
          )}
        </div>
        {followError ? <p className="mt-4 text-sm text-rose-600">{followError}</p> : null}

        {canViewFullProfile && (
          <div className="mt-8 grid gap-4 sm:grid-cols-3">
            <StatButton
              label="Followers"
              value={profile.follower_count}
              onClick={() => openStatsPanel('followers')}
            />
            <StatButton
              label="Following"
              value={profile.following_count}
              onClick={() => openStatsPanel('following')}
            />
            <StatButton
              label="Posts"
              value={profile.post_count}
              onClick={() => openStatsPanel('posts')}
            />
          </div>
        )}
      </div>

      {isPrivateLimited ? (
        <div className="rounded-xl border border-gray-100 bg-white p-8 text-center">
          <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-xl bg-amber-50 text-amber-600">
            <Lock className="h-6 w-6" />
          </div>
          <h2 className="mt-4 text-xl font-semibold text-gray-900">This profile is private</h2>
          <p className="mx-auto mt-2 max-w-md text-sm leading-6 text-gray-500">
            Follow this person to see their posts, followers, following, and profile details.
          </p>
        </div>
      ) : (
        <div className="grid gap-4 lg:grid-cols-3">
          <div className="lg:col-span-2 space-y-4">
            <div className="rounded-xl border border-gray-100 bg-white p-6">
              <div className="flex items-center justify-between">
                <div>
                  <h2 className="text-xl font-semibold text-gray-900">About</h2>
                </div>
                <BookUser className="w-5 h-5 text-indigo-500" />
              </div>
              <p className="mt-6 text-gray-600 leading-7">
                {profile.about_me?.trim() || 'No about section has been added yet.'}
              </p>
            </div>

            <div className="rounded-xl border border-gray-100 bg-white p-6">
              <h2 className="text-xl font-semibold text-gray-900">Profile details</h2>
              <div className="mt-6 space-y-4">
                <DetailRow label="Full name" value={`${profile.user.first_name} ${profile.user.last_name}`} />
                <DetailRow label="Nickname" value={profile.user.nickname || 'Not set'} />
                {email && <DetailRow label="Email" value={email} icon={<Mail className="h-4 w-4" />} />}
                <DetailRow label="Birthday" value={formatDate(profile.dob)} icon={<Calendar className="h-4 w-4" />} />
              </div>
            </div>
          </div>

          <div className="space-y-4">
            <div className="rounded-xl border border-gray-100 bg-white p-6">
              <div className="flex items-center justify-between">
                <div>
                  <h2 className="text-xl font-semibold text-gray-900">Profile summary</h2>
                </div>
                <Users className="w-5 h-5 text-indigo-500" />
              </div>

              <div className="mt-6 grid gap-4">
                <div className="rounded-xl border border-gray-100 bg-gray-50 p-4">
                  <p className="text-sm text-gray-500">Visibility</p>
                  <p className="mt-2 font-semibold text-gray-900">
                    {profile.user.is_public ? 'Open to everyone' : 'Only visible to followers'}
                  </p>
                </div>
                <div className="rounded-xl border border-gray-100 bg-gray-50 p-4">
                  <p className="text-sm text-gray-500">Birthday on file</p>
                  <p className="mt-2 font-semibold text-gray-900">
                    {profile.dob ? formatDate(profile.dob) : 'Date not available'}
                  </p>
                </div>
                {isOwnProfile && (
                  <div className="rounded-xl border border-gray-100 bg-gray-50 p-4">
                    <p className="text-sm text-gray-500">Owner status</p>
                    <p className="mt-2 inline-flex items-center gap-2 font-semibold text-gray-900">
                      <CheckCircle2 className="h-4 w-4 text-emerald-500" />
                      This is your profile
                    </p>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      {panel && <StatsPanelDialog panel={panel} onClose={() => setPanel(null)} />}
    </div>
  );
}

function LabeledField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <span className="mb-2 block text-sm font-medium text-gray-700">{label}</span>
      {children}
    </label>
  );
}

function StatButton({
  label,
  value,
  onClick,
}: {
  label: string;
  value: number;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="rounded-xl border border-gray-100 bg-gray-50 p-5 text-left transition-colors hover:border-indigo-100 hover:bg-indigo-50/40 focus:outline-none focus:ring-2 focus:ring-indigo-500"
    >
      <p className="text-sm text-gray-500">{label}</p>
      <p className="mt-2 text-3xl font-semibold text-gray-900">{value.toLocaleString()}</p>
    </button>
  );
}

function DetailRow({
  label,
  value,
  icon,
}: {
  label: string;
  value: string;
  icon?: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-2 rounded-xl border border-gray-100 bg-gray-50 p-4 sm:flex-row sm:items-center sm:justify-between">
      <span className="text-sm text-gray-500">{label}</span>
      <span className="inline-flex items-center gap-2 font-medium text-gray-900">
        {icon}
        {value}
      </span>
    </div>
  );
}

function StatsPanelDialog({
  panel,
  onClose,
}: {
  panel: StatsPanelState;
  onClose: () => void;
}) {
  const title =
    panel.type === 'followers' ? 'Followers' : panel.type === 'following' ? 'Following' : 'Posts';
  const hasItems = panel.type === 'posts' ? panel.posts.length > 0 : panel.users.length > 0;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 px-4 py-6">
      <div className="flex max-h-[80vh] w-full max-w-2xl flex-col overflow-hidden rounded-xl border border-gray-100 bg-white shadow-xl">
        <div className="flex items-center justify-between border-b border-gray-100 px-6 py-4">
          <div>
            <h2 className="text-lg font-semibold text-gray-900">{title}</h2>
            <p className="text-sm text-gray-500">Profile {title.toLowerCase()}</p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-full p-2 text-gray-500 hover:bg-gray-100 hover:text-gray-700"
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="overflow-y-auto p-6">
          {panel.loading && (
            <div className="space-y-3">
              {[1, 2, 3].map((item) => (
                <div key={item} className="h-16 animate-pulse rounded-xl bg-gray-100" />
              ))}
            </div>
          )}

          {!panel.loading && panel.error && (
            <div className="rounded-xl border border-red-100 bg-red-50 p-4 text-sm text-red-600">
              {panel.error}
            </div>
          )}

          {!panel.loading && !panel.error && !hasItems && (
            <div className="rounded-xl border border-gray-100 bg-gray-50 p-8 text-center">
              <p className="text-sm text-gray-500">Nothing to show yet.</p>
            </div>
          )}

          {!panel.loading && !panel.error && panel.type !== 'posts' && (
            <div className="space-y-3">
              {panel.users.map((user) => (
                <div
                  key={user.id}
                  className="flex items-center gap-3 rounded-xl border border-gray-100 bg-gray-50 p-3"
                >
                  <Link href={`/profile/${user.id}`} className="contents">
                    <Avatar user={user} />
                    <div className="min-w-0">
                      <p className="truncate font-semibold text-gray-900 hover:text-indigo-600">{getDisplayName(user)}</p>
                      <p className="truncate text-sm text-gray-500">
                        @{user.first_name.toLowerCase()}{user.last_name.toLowerCase()}
                      </p>
                    </div>
                  </Link>
                </div>
              ))}
            </div>
          )}

          {!panel.loading && !panel.error && panel.type === 'posts' && (
            <div className="space-y-3">
              {panel.posts.map((post) => (
                <ProfilePostCard key={post.id} post={post} />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function Avatar({ user }: { user: PublicUser }) {
  const avatarUrl = resolveAssetUrl(user.avatar_path);

  if (avatarUrl) {
    return <img src={avatarUrl} alt={getDisplayName(user)} className="h-11 w-11 rounded-full object-cover" />;
  }

  return (
    <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-indigo-100 text-sm font-semibold text-indigo-700">
      {getInitials(user)}
    </div>
  );
}

function ProfilePostCard({ post }: { post: PostResponse }) {
  const postImageUrl = resolveAssetUrl(post.image_url);

  return (
    <div className="overflow-hidden rounded-xl border border-gray-100 bg-white">
      <div className="flex items-center justify-between p-4">
        <div className="flex items-center gap-3">
          <Link href={`/profile/${post.author.id}`} className="contents">
            <Avatar user={post.author} />
            <div>
              <p className="text-sm font-bold text-gray-900 hover:text-indigo-600">
                {post.author.first_name} {post.author.last_name}
              </p>
              <p className="text-xs text-gray-500">
                {formatShortDate(post.created_at)}{' '}
                {post.privacy ? `• ${formatPostPrivacyLabel(post.privacy)}` : ''}
              </p>
            </div>
          </Link>
        </div>
        <button type="button" className="text-gray-400 hover:text-gray-600" aria-label="Post options">
          <MoreHorizontal className="h-5 w-5" />
        </button>
      </div>

      {post.content ? (
        <div className="px-4 pb-4">
          <p className="whitespace-pre-wrap text-sm leading-relaxed text-gray-800">{post.content}</p>
        </div>
      ) : null}

      {postImageUrl ? (
        <div className="px-4 pb-4">
          <img
            src={postImageUrl}
            alt="Post image"
            className="w-full max-h-[32rem] rounded-xl border border-gray-100 bg-gray-50 object-cover"
          />
        </div>
      ) : null}
    </div>
  );
}

export default ProfilePageView;
