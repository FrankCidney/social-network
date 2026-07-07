'use client';

import React from 'react';
import Link from 'next/link';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { Home, Users, MessageSquare, Bell, User, LogOut, Search } from 'lucide-react';
import { PeopleDiscoveryPanel } from '@/components/discovery/PeopleDiscoveryPanel';
import { WebSocketProvider } from '@/contexts/WebSocketContext';
import { NotificationsProvider } from '@/contexts/NotificationsContext';
import { MessageUnreadProvider, useMessageUnread } from '@/contexts/MessageUnreadContext';
import { NotificationDropdown } from '@/components/notifications/NotificationDropdown';
import { authAPI, isAuthenticationError, profileAPI, resolveAssetUrl, type PublicUser } from '@/lib/api';

export default function MainLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();
  const isGroupsRoute = pathname.startsWith('/groups');
  const [authChecked, setAuthChecked] = React.useState(false);
  const [authError, setAuthError] = React.useState<string | null>(null);
  const [isLoggingOut, setIsLoggingOut] = React.useState(false);
  const [currentUser, setCurrentUser] = React.useState<PublicUser | null>(null);

  React.useEffect(() => {
    let cancelled = false;

    const verifySession = async () => {
      setAuthChecked(false);
      setAuthError(null);

      try {
        const profile = await profileAPI.getMyProfile();
        if (!cancelled) {
          setCurrentUser(profile.user);
          setAuthChecked(true);
        }
      } catch (error) {
        if (cancelled) {
          return;
        }

        if (isAuthenticationError(error)) {
          const queryString = searchParams.toString();
          const nextPath = `${pathname}${queryString ? `?${queryString}` : ''}`;
          router.replace(`/login?next=${encodeURIComponent(nextPath)}`);
          return;
        }

        setCurrentUser(null);
        setAuthError(
          error instanceof Error ? error.message : 'Unable to verify your session.'
        );
        setAuthChecked(true);
      }
    };

    void verifySession();

    return () => {
      cancelled = true;
    };
  }, [pathname, router, searchParams]);

  const handleLogout = React.useCallback(async () => {
    setIsLoggingOut(true);
    setAuthError(null);

    try {
      await authAPI.logout();
    } catch (error) {
      if (!isAuthenticationError(error)) {
        setAuthError(
          error instanceof Error ? error.message : 'Unable to log out right now.'
        );
        setIsLoggingOut(false);
        return;
      }
    }

    router.replace('/login');
    router.refresh();
  }, [router]);

  const topBarAvatarUrl = resolveAssetUrl(currentUser?.avatar_path);
  const topBarInitials = currentUser
    ? `${currentUser.first_name?.[0] ?? ''}${currentUser.last_name?.[0] ?? ''}`.trim().toUpperCase() || 'Y'
    : 'Y';

  if (!authChecked) {
    return (
      <div className="min-h-screen bg-gray-50">
        <div className="mx-auto flex min-h-screen max-w-7xl items-center justify-center px-4">
          <div className="rounded-xl border border-gray-100 bg-white px-6 py-4 text-sm text-gray-500 shadow-sm">
            Checking your session...
          </div>
        </div>
      </div>
    );
  }

  if (authError) {
    return (
      <div className="min-h-screen bg-gray-50">
        <div className="mx-auto flex min-h-screen max-w-7xl items-center justify-center px-4">
          <div className="rounded-xl border border-red-200 bg-white px-6 py-5 text-center">
            <p className="text-sm text-red-600">{authError}</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <WebSocketProvider>
      <MessageUnreadProvider>
        <NotificationsProvider>
          <MainShell
            currentUser={currentUser}
            handleLogout={handleLogout}
            isGroupsRoute={isGroupsRoute}
            isLoggingOut={isLoggingOut}
            pathname={pathname}
            topBarAvatarUrl={topBarAvatarUrl}
            topBarInitials={topBarInitials}
          >
            {children}
          </MainShell>
        </NotificationsProvider>
      </MessageUnreadProvider>
    </WebSocketProvider>
  );
}

function MainShell({
  children,
  handleLogout,
  isGroupsRoute,
  isLoggingOut,
  pathname,
  topBarAvatarUrl,
  topBarInitials,
}: {
  children: React.ReactNode;
  currentUser: PublicUser | null;
  handleLogout: () => void;
  isGroupsRoute: boolean;
  isLoggingOut: boolean;
  pathname: string;
  topBarAvatarUrl?: string;
  topBarInitials: string;
}) {
  const { unreadCount: unreadMessageCount } = useMessageUnread();

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Navigation Bar */}
      <nav className="sticky top-0 z-50 bg-white border-b border-gray-100">
        <div className="max-w-7xl mx-auto px-4 h-16 flex items-center justify-between">
          <div className="flex items-center gap-8">
            <Link href="/feed" className="text-2xl font-bold text-indigo-600">Social</Link>
            <div className="hidden md:flex relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                type="text"
                placeholder="Search social..."
                className="bg-gray-100 rounded-full py-2 pl-10 pr-4 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 w-64 transition-all"
              />
            </div>
          </div>

          <div className="flex items-center gap-2">
            <NotificationDropdown />
            <Link href="/profile" className="flex items-center gap-2 p-1 pl-3 rounded-full hover:bg-gray-100 transition-colors">
              <span className="text-sm font-medium hidden sm:inline">My Profile</span>
              {topBarAvatarUrl ? (
                <img
                  src={topBarAvatarUrl}
                  alt="My profile avatar"
                  className="h-8 w-8 rounded-full object-cover"
                />
              ) : (
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-indigo-100 text-xs font-semibold text-indigo-700">
                  {topBarInitials}
                </div>
              )}
            </Link>
          </div>
        </div>
      </nav>

      <div className="max-w-7xl mx-auto grid grid-cols-1 gap-6 px-4 pb-24 pt-6 md:grid-cols-12 md:pb-6">
        {/* Sidebar */}
        <aside className="hidden md:block md:col-span-3 space-y-2">
          <SidebarItem icon={<Home className="w-5 h-5" />} label="Home Feed" href="/feed" active={pathname === '/feed'} />
          <SidebarItem icon={<Users className="w-5 h-5" />} label="Groups" href="/groups" active={pathname.startsWith('/groups')} />
          <SidebarItem icon={<MessageSquare className="w-5 h-5" />} label="Messages" href="/messages" active={pathname === '/messages'} badgeCount={unreadMessageCount} />
          <SidebarItem icon={<Bell className="w-5 h-5" />} label="Notifications" href="/notifications" active={pathname === '/notifications'} />
          <SidebarItem icon={<User className="w-5 h-5" />} label="Profile" href="/profile" active={pathname === '/profile'} />
          <hr className="my-4 border-gray-100" />
          <SidebarAction
            icon={<LogOut className="w-5 h-5" />}
            label={isLoggingOut ? 'Logging out...' : 'Logout'}
            onClick={handleLogout}
            disabled={isLoggingOut}
          />
        </aside>

        {/* Main Content */}
        <main className={isGroupsRoute ? 'md:col-span-9' : 'md:col-span-9 lg:col-span-6'}>
          {children}
        </main>

        {/* Right Sidebar (Suggestions/Trends) */}
        {!isGroupsRoute && (
          <>
            <aside className="hidden lg:block lg:col-span-3 space-y-6">
              <PeopleDiscoveryPanel />
            </aside>
            <aside className="md:col-start-4 md:col-span-9 lg:hidden">
              <PeopleDiscoveryPanel />
            </aside>
          </>
        )}
      </div>

      <nav className="fixed inset-x-0 bottom-0 z-50 border-t border-gray-200 bg-white/95 px-2 py-2 shadow-[0_-8px_24px_rgba(15,23,42,0.08)] backdrop-blur md:hidden">
        <div className="mx-auto grid max-w-md grid-cols-5 gap-1">
          <MobileNavItem icon={<Home className="h-5 w-5" />} label="Feed" href="/feed" active={pathname === '/feed'} />
          <MobileNavItem icon={<Users className="h-5 w-5" />} label="Groups" href="/groups" active={pathname.startsWith('/groups')} />
          <MobileNavItem icon={<MessageSquare className="h-5 w-5" />} label="Messages" href="/messages" active={pathname === '/messages'} badgeCount={unreadMessageCount} />
          <MobileNavItem icon={<Bell className="h-5 w-5" />} label="Alerts" href="/notifications" active={pathname === '/notifications'} />
          <MobileNavItem icon={<User className="h-5 w-5" />} label="Profile" href="/profile" active={pathname === '/profile'} />
        </div>
      </nav>
    </div>
  );
}

function SidebarItem({ icon, label, href, active = false, badgeCount = 0 }: { icon: React.ReactNode, label: string, href: string, active?: boolean, badgeCount?: number }) {
  return (
    <Link 
      href={href}
      className={`flex items-center gap-3 px-4 py-3 rounded-xl transition-colors font-medium ${
        active 
          ? 'bg-indigo-50 text-indigo-700' 
          : 'text-gray-600 hover:bg-gray-100'
      }`}
    >
      {icon}
      <span>{label}</span>
      {badgeCount > 0 ? <UnreadBadge count={badgeCount} className="ml-auto" /> : null}
    </Link>
  );
}

function MobileNavItem({ icon, label, href, active = false, badgeCount = 0 }: { icon: React.ReactNode, label: string, href: string, active?: boolean, badgeCount?: number }) {
  return (
    <Link
      href={href}
      className={`flex min-h-12 flex-col items-center justify-center gap-1 rounded-xl px-1 text-[11px] font-semibold transition-colors ${
        active
          ? 'bg-indigo-50 text-indigo-700'
          : 'text-gray-500 hover:bg-gray-100 hover:text-gray-700'
      }`}
    >
      <span className="relative">
        {icon}
        {badgeCount > 0 ? <UnreadBadge count={badgeCount} className="absolute -right-3 -top-2" /> : null}
      </span>
      <span className="max-w-full truncate">{label}</span>
    </Link>
  );
}

function UnreadBadge({ count, className = '' }: { count: number; className?: string }) {
  return (
    <span className={`inline-flex min-w-5 items-center justify-center rounded-full bg-red-500 px-1.5 py-0.5 text-[10px] font-semibold leading-none text-white ${className}`}>
      {count > 9 ? '9+' : count}
    </span>
  );
}

function SidebarAction({
  icon,
  label,
  onClick,
  disabled = false,
}: {
  icon: React.ReactNode;
  label: string;
  onClick: () => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className="flex w-full items-center gap-3 rounded-xl px-4 py-3 text-left font-medium text-gray-600 transition-colors hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-60"
    >
      {icon}
      <span>{label}</span>
    </button>
  );
}
