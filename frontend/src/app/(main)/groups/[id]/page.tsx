'use client';

import * as React from 'react';
import Image from 'next/image';
import { useParams } from 'next/navigation';
import {
	Calendar,
	CalendarPlus,
	Loader,
	MessageSquare,
	Check,
	Send,
	Sparkles,
	UserPlus,
	Users,
	X,
} from 'lucide-react';
import { EmojiPicker } from '@/components/chat/EmojiPicker';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import {
	type GroupDetail,
	type GroupEvent,
	type GroupMessage,
	type PostResponse,
	type ProfileResponse,
	type PublicUser,
	chatAPI,
	feedAPI,
	groupAPI,
	profileAPI,
	resolveAssetUrl,
} from '@/lib/api';

const GROUP_POST_LIMIT = 20;
const GROUP_MESSAGE_LIMIT = 50;

type SideTab = 'chat' | 'events' | 'members' | 'manage';

type GroupWorkspaceState = {
	group: GroupDetail | null;
	profile: ProfileResponse | null;
	events: GroupEvent[];
	posts: PostResponse[];
	messages: GroupMessage[];
};

const INITIAL_STATE: GroupWorkspaceState = {
	group: null,
	profile: null,
	events: [],
	posts: [],
	messages: [],
};

export default function GroupDetailPage() {
	const params = useParams<{ id: string }>();
	const groupId = Array.isArray(params.id) ? params.id[0] : params.id;
	const [workspace, setWorkspace] = React.useState<GroupWorkspaceState>(INITIAL_STATE);
	const [loading, setLoading] = React.useState(true);
	const [pageError, setPageError] = React.useState<string | null>(null);
	const [postsError, setPostsError] = React.useState<string | null>(null);
	const [messagesError, setMessagesError] = React.useState<string | null>(null);
	const [membershipNotice, setMembershipNotice] = React.useState<string | null>(null);
	const [joinLoading, setJoinLoading] = React.useState(false);
	const [postDraft, setPostDraft] = React.useState('');
	const [postError, setPostError] = React.useState<string | null>(null);
	const [postSubmitting, setPostSubmitting] = React.useState(false);
	const [messageDraft, setMessageDraft] = React.useState('');
	const [messageSubmitting, setMessageSubmitting] = React.useState(false);
	const [activeTab, setActiveTab] = React.useState<SideTab>('chat');
	const [eventFormOpen, setEventFormOpen] = React.useState(false);
	const [eventTitle, setEventTitle] = React.useState('');
	const [eventDescription, setEventDescription] = React.useState('');
	const [eventDate, setEventDate] = React.useState('');
	const [eventError, setEventError] = React.useState<string | null>(null);
	const [eventSubmitting, setEventSubmitting] = React.useState(false);
	const [members, setMembers] = React.useState<PublicUser[]>([]);
	const [membersLoading, setMembersLoading] = React.useState(false);
	const [membersError, setMembersError] = React.useState<string | null>(null);
	const [requests, setRequests] = React.useState<PublicUser[]>([]);
	const [requestsLoading, setRequestsLoading] = React.useState(false);
	const [requestsError, setRequestsError] = React.useState<string | null>(null);
	const [inviteQuery, setInviteQuery] = React.useState('');
	const deferredInviteQuery = React.useDeferredValue(inviteQuery);
	const [inviteResults, setInviteResults] = React.useState<PublicUser[]>([]);
	const [inviteSearchLoading, setInviteSearchLoading] = React.useState(false);
	const [inviteSubmitting, setInviteSubmitting] = React.useState(false);
	const [inviteError, setInviteError] = React.useState<string | null>(null);
	const [eventResponses, setEventResponses] = React.useState<Record<string, 'going' | 'not_going'>>({});
	const messageInputRef = React.useRef<HTMLInputElement>(null);

	React.useEffect(() => {
		if (!groupId) {
			return;
		}

		const loadWorkspace = async () => {
			setLoading(true);
			setPageError(null);
			setPostsError(null);
			setMessagesError(null);
			setMembershipNotice(null);

			try {
				const [groupResult, profileResult, eventsResult, postsResult, messagesResult] =
					await Promise.allSettled([
						groupAPI.getGroup(groupId),
						profileAPI.getMyProfile(),
						groupAPI.getEvents(groupId),
						feedAPI.getGroupPosts(groupId, GROUP_POST_LIMIT, 0),
						chatAPI.getGroupMessages(groupId, GROUP_MESSAGE_LIMIT, 0),
					]);

				if (groupResult.status !== 'fulfilled') {
					throw groupResult.reason;
				}

				const nextState: GroupWorkspaceState = {
					group: groupResult.value,
					profile: profileResult.status === 'fulfilled' ? profileResult.value : null,
					events:
						eventsResult.status === 'fulfilled' && Array.isArray(eventsResult.value)
							? eventsResult.value
							: [],
					posts:
						postsResult.status === 'fulfilled' ? postsResult.value.posts ?? [] : [],
					messages:
						messagesResult.status === 'fulfilled' && Array.isArray(messagesResult.value)
							? messagesResult.value
							: [],
				};

				if (postsResult.status === 'rejected') {
					setPostsError(extractError(postsResult.reason, 'Group discussion is available to members only right now.'));
				}

				if (messagesResult.status === 'rejected') {
					setMessagesError(extractError(messagesResult.reason, 'Group chat is available to members only right now.'));
				}

				setWorkspace(nextState);
			} catch (err) {
				setPageError(err instanceof Error ? err.message : 'Failed to load group');
			} finally {
				setLoading(false);
			}
		};

		void loadWorkspace();
	}, [groupId]);

	const group = workspace.group;
	const currentUser = workspace.profile?.user ?? null;
	const isCreator = Boolean(group?.is_creator);
	const membershipStatus = group?.membership_status ?? 'none';
	const canInvite = membershipStatus === 'accepted' || isCreator;
	const canModerateRequests = isCreator;

	React.useEffect(() => {
		if (!groupId || activeTab !== 'members' || membershipStatus !== 'accepted') {
			return;
		}

		const loadMembers = async () => {
			setMembersLoading(true);
			setMembersError(null);
			try {
				const response = await groupAPI.getMembers(groupId);
				setMembers(Array.isArray(response) ? response : []);
			} catch (err) {
				setMembersError(err instanceof Error ? err.message : 'Failed to load members');
			} finally {
				setMembersLoading(false);
			}
		};

		void loadMembers();
	}, [activeTab, groupId, membershipStatus]);

	React.useEffect(() => {
		if (!groupId || activeTab !== 'manage' || !canModerateRequests) {
			return;
		}

		const loadRequests = async () => {
			setRequestsLoading(true);
			setRequestsError(null);
			try {
				const response = await groupAPI.getJoinRequests(groupId);
				setRequests(Array.isArray(response) ? response : []);
			} catch (err) {
				setRequestsError(err instanceof Error ? err.message : 'Failed to load requests');
			} finally {
				setRequestsLoading(false);
			}
		};

		void loadRequests();
	}, [activeTab, canModerateRequests, groupId]);

	React.useEffect(() => {
		if (!groupId || activeTab !== 'manage' || !canInvite) {
			setInviteResults([]);
			setInviteSearchLoading(false);
			return;
		}

		const query = deferredInviteQuery.trim();
		if (query.length < 2) {
			setInviteResults([]);
			setInviteSearchLoading(false);
			setInviteError(null);
			return;
		}

		let cancelled = false;

		const loadInviteResults = async () => {
			setInviteSearchLoading(true);
			setInviteError(null);
			try {
				const response = await profileAPI.searchUsers(query, {
					limit: 8,
					exclude_group_id: groupId,
				});
				if (!cancelled) {
					setInviteResults(Array.isArray(response) ? response : []);
				}
			} catch (err) {
				if (!cancelled) {
					setInviteResults([]);
					setInviteError(err instanceof Error ? err.message : 'Failed to search users');
				}
			} finally {
				if (!cancelled) {
					setInviteSearchLoading(false);
				}
			}
		};

		void loadInviteResults();

		return () => {
			cancelled = true;
		};
	}, [activeTab, canInvite, deferredInviteQuery, groupId]);

	const handleJoinRequest = async () => {
		if (!groupId) {
			return;
		}

		setJoinLoading(true);
		setMembershipNotice(null);

		try {
			const response = await groupAPI.requestJoin(groupId);
			setMembershipNotice(response.message || 'Join request sent.');
			setWorkspace((current) =>
				current.group
					? { ...current, group: { ...current.group, membership_status: 'requested' } }
					: current
			);
		} catch (err) {
			setMembershipNotice(err instanceof Error ? err.message : 'Unable to send join request');
		} finally {
			setJoinLoading(false);
		}
	};

	const handleInviteDecision = async (accept: boolean) => {
		if (!groupId) {
			return;
		}

		setJoinLoading(true);
		setMembershipNotice(null);

		try {
			if (accept) {
				await groupAPI.acceptInvite(groupId);
				setWorkspace((current) =>
					current.group
						? { ...current, group: { ...current.group, membership_status: 'accepted' } }
						: current
				);
				setMembershipNotice('Invite accepted.');
			} else {
				await groupAPI.declineInvite(groupId);
				setWorkspace((current) =>
					current.group
						? { ...current, group: { ...current.group, membership_status: 'declined' } }
						: current
				);
				setMembershipNotice('Invite declined.');
			}
		} catch (err) {
			setMembershipNotice(err instanceof Error ? err.message : 'Unable to update invite');
		} finally {
			setJoinLoading(false);
		}
	};

	const handleCreatePost = async (event: React.FormEvent<HTMLFormElement>) => {
		event.preventDefault();

		if (!groupId) {
			return;
		}

		const content = postDraft.trim();
		if (!content) {
			setPostError('Write something before posting to the group.');
			return;
		}

		setPostSubmitting(true);
		setPostError(null);

		try {
			const createdPost = await feedAPI.createPost({
				content,
				privacy: 'group',
				group_id: groupId,
			});
			const hydratedPost = await feedAPI.getPost(createdPost.id);
			setWorkspace((current) => ({
				...current,
				posts: [hydratedPost, ...current.posts],
			}));
			setPostDraft('');
		} catch (err) {
			setPostError(err instanceof Error ? err.message : 'Failed to create group post');
		} finally {
			setPostSubmitting(false);
		}
	};

	const handleSendMessage = async (event: React.FormEvent<HTMLFormElement>) => {
		event.preventDefault();

		if (!groupId) {
			return;
		}

		const content = messageDraft.trim();
		if (!content) {
			return;
		}

		setMessageSubmitting(true);
		setMessagesError(null);

		try {
			const sentMessage = await chatAPI.sendMessage({
				group_id: groupId,
				content,
			});
			setWorkspace((current) => ({
				...current,
				messages: [
					...current.messages,
					{
						id: sentMessage.id,
						sender_id: sentMessage.sender_id,
						group_id: groupId,
						content: sentMessage.content,
						created_at: sentMessage.created_at,
					},
				],
			}));
			setMessageDraft('');
		} catch (err) {
			setMessagesError(err instanceof Error ? err.message : 'Failed to send group message');
		} finally {
			setMessageSubmitting(false);
		}
	};

	const insertMessageEmoji = (emoji: string) => {
		const input = messageInputRef.current;
		const start = input?.selectionStart ?? messageDraft.length;
		const end = input?.selectionEnd ?? messageDraft.length;
		const nextDraft = `${messageDraft.slice(0, start)}${emoji}${messageDraft.slice(end)}`;

		setMessageDraft(nextDraft);

		requestAnimationFrame(() => {
			input?.focus();
			const nextCursor = start + emoji.length;
			input?.setSelectionRange(nextCursor, nextCursor);
		});
	};

	const handleCreateEvent = async (event: React.FormEvent<HTMLFormElement>) => {
		event.preventDefault();

		if (!groupId) {
			return;
		}

		if (!eventTitle.trim() || !eventDate) {
			setEventError('Add a title and date/time before creating the event.');
			return;
		}

		setEventSubmitting(true);
		setEventError(null);

		try {
			const createdEvent = await groupAPI.createEvent(groupId, {
				title: eventTitle.trim(),
				description: eventDescription.trim(),
				event_date: new Date(eventDate).toISOString(),
			});
			setWorkspace((current) => ({
				...current,
				events: [createdEvent, ...current.events].sort(
					(a, b) => new Date(a.event_date).getTime() - new Date(b.event_date).getTime()
				),
			}));
			setEventTitle('');
			setEventDescription('');
			setEventDate('');
			setEventFormOpen(false);
		} catch (err) {
			setEventError(err instanceof Error ? err.message : 'Failed to create event');
		} finally {
			setEventSubmitting(false);
		}
	};

	const handleInviteUser = async (invitee: PublicUser) => {
		if (!groupId) {
			return;
		}

		setInviteSubmitting(true);
		setInviteError(null);
		try {
			await groupAPI.inviteUser(groupId, { invitee_id: invitee.id });
			setInviteResults((current) => current.filter((user) => user.id !== invitee.id));
			setInviteQuery('');
			setMembershipNotice('Invite sent.');
		} catch (err) {
			setInviteError(err instanceof Error ? err.message : 'Failed to send invite');
		} finally {
			setInviteSubmitting(false);
		}
	};

	const handleJoinRequestDecision = async (userId: string, accept: boolean) => {
		if (!groupId) {
			return;
		}

		try {
			if (accept) {
				await groupAPI.acceptJoinRequest(groupId, userId);
			} else {
				await groupAPI.declineJoinRequest(groupId, userId);
			}
			setRequests((current) => current.filter((user) => user.id !== userId));
		} catch (err) {
			setRequestsError(err instanceof Error ? err.message : 'Failed to update request');
		}
	};

	const handleEventResponse = async (eventId: string, status: 'going' | 'not_going') => {
		try {
			await groupAPI.rsvpEvent(eventId, status);
			setEventResponses((current) => ({ ...current, [eventId]: status }));
		} catch (err) {
			setEventError(err instanceof Error ? err.message : 'Failed to update RSVP');
		}
	};

	if (loading) {
		return (
			<div className="flex min-h-[28rem] items-center justify-center rounded-[30px] border border-gray-100 bg-white">
				<Loader className="h-8 w-8 animate-spin text-indigo-600" />
			</div>
		);
	}

	if (pageError || !group) {
		return (
			<div className="rounded-[30px] border border-red-200 bg-red-50 p-6 text-red-600">
				<p className="font-bold">Error loading group</p>
				<p className="mt-1 text-sm">{pageError ?? 'Group not found'}</p>
			</div>
		);
	}

	return (
		<div className="space-y-6">
			<section className="rounded-[30px] border border-gray-100 bg-white px-6 py-7 lg:px-8">
				<div className="flex flex-wrap items-start justify-between gap-4">
						<div className="space-y-3">
							<h1 className="text-3xl font-bold tracking-tight text-gray-900">{group.title}</h1>
							{group.description ? (
								<p className="max-w-2xl text-sm leading-6 text-gray-600">{group.description}</p>
							) : null}
							<div className="flex flex-wrap items-center gap-2 text-sm text-gray-500">
								<MetaPill label={`${group.creator.first_name} ${group.creator.last_name}`} />
								<MetaPill label={new Date(group.created_at).toLocaleDateString()} />
							</div>
						</div>
						{membershipStatus === 'none' ? (
							<Button type="button" onClick={handleJoinRequest} disabled={joinLoading}>
								{joinLoading ? <Loader className="mr-2 h-4 w-4 animate-spin" /> : <UserPlus className="mr-2 h-4 w-4" />}
								Request to Join
							</Button>
						) : null}
						{membershipStatus === 'invited' ? (
							<div className="flex flex-wrap gap-2">
								<Button type="button" onClick={() => void handleInviteDecision(true)} disabled={joinLoading}>
									{joinLoading ? <Loader className="mr-2 h-4 w-4 animate-spin" /> : <Check className="mr-2 h-4 w-4" />}
									Accept
								</Button>
								<Button type="button" variant="secondary" onClick={() => void handleInviteDecision(false)} disabled={joinLoading}>
									<X className="mr-2 h-4 w-4" />
									Decline
								</Button>
							</div>
						) : null}
						{membershipStatus === 'requested' ? (
							<MetaPill label="Request sent" />
						) : null}
					</div>
				{membershipNotice ? (
					<div className="mt-5 rounded-[18px] border border-indigo-100 bg-indigo-50 px-4 py-3 text-sm text-indigo-700">
						{membershipNotice}
					</div>
				) : null}
			</section>

			<div className="grid gap-6 xl:grid-cols-[minmax(0,1.65fr)_minmax(21rem,0.95fr)]">
				<div className="space-y-6">
					<section className="space-y-4">
						{postsError ? (
							<UnavailableState description={postsError} />
						) : (
							<>
								<form className="rounded-xl border border-gray-100 bg-white p-4" onSubmit={handleCreatePost}>
									<div className="flex gap-3">
										{renderAvatar(currentUser ?? undefined)}
										<div className="flex-1 space-y-3">
											<textarea
												value={postDraft}
												onChange={(event) => setPostDraft(event.target.value)}
												placeholder="Share something with the group..."
												rows={3}
												className="w-full resize-none rounded-xl border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-800 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
											/>
											<div className="flex flex-wrap items-center justify-between gap-3">
												{postError ? <p className="text-sm text-rose-600">{postError}</p> : <span />}
												<Button type="submit" disabled={postSubmitting}>
													{postSubmitting ? <Loader className="mr-2 h-4 w-4 animate-spin" /> : <Sparkles className="mr-2 h-4 w-4" />}
													Post
												</Button>
											</div>
										</div>
									</div>
								</form>

								{workspace.posts.length > 0 ? (
									workspace.posts.map((post) => (
										<GroupPostCard key={post.id} post={post} />
									))
								) : (
									<EmptyState title="No posts yet" />
								)}
							</>
						)}
					</section>
				</div>

				<section className="rounded-[28px] border border-gray-100 bg-white p-4 shadow-sm">
					<div className="grid grid-cols-2 gap-2">
						<TabButton label="Chat" active={activeTab === 'chat'} onClick={() => setActiveTab('chat')} />
						<TabButton label="Events" active={activeTab === 'events'} onClick={() => setActiveTab('events')} />
						<TabButton label="Members" active={activeTab === 'members'} onClick={() => setActiveTab('members')} />
						<TabButton label="Manage" active={activeTab === 'manage'} onClick={() => setActiveTab('manage')} />
					</div>

					<div className="mt-4">
						{activeTab === 'chat' ? (
							messagesError ? (
								<UnavailableState description={messagesError} />
							) : (
								<div className="space-y-4">
									<div className="max-h-[30rem] space-y-3 overflow-y-auto rounded-[24px] border border-gray-100 bg-gray-50/80 p-4">
										{workspace.messages.length > 0 ? (
											workspace.messages.map((message) => {
												const isOwnMessage = currentUser?.id === message.sender_id;
												return (
													<div
														key={message.id}
														className={`max-w-[85%] rounded-[20px] px-4 py-3 text-sm ${
															isOwnMessage ? 'ml-auto bg-indigo-600 text-white' : 'bg-white text-gray-700 shadow-sm'
														}`}
													>
														<p className={`text-xs font-semibold ${isOwnMessage ? 'text-indigo-100' : 'text-gray-500'}`}>
															{isOwnMessage ? 'You' : 'Member'}
														</p>
														<p className="mt-1 whitespace-pre-wrap leading-6">{message.content}</p>
														<p className={`mt-2 text-[11px] ${isOwnMessage ? 'text-indigo-100' : 'text-gray-400'}`}>
															{formatDateTime(message.created_at)}
														</p>
													</div>
												);
											})
										) : (
											<EmptyState title="No messages yet" />
										)}
									</div>

									<form className="flex gap-3" onSubmit={handleSendMessage}>
										<div className="relative flex-1">
											<input
												ref={messageInputRef}
												type="text"
												value={messageDraft}
												onChange={(event) => setMessageDraft(event.target.value)}
												placeholder="Send a message..."
												disabled={messageSubmitting}
												className="w-full rounded-full border border-gray-200 bg-white py-3 pl-4 pr-12 text-sm text-gray-900 placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:opacity-60"
											/>
											<div className="absolute right-1.5 top-1/2 -translate-y-1/2">
												<EmojiPicker onSelect={insertMessageEmoji} disabled={messageSubmitting} />
											</div>
										</div>
										<Button type="submit" disabled={messageSubmitting}>
											{messageSubmitting ? <Loader className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
										</Button>
									</form>
								</div>
							)
						) : null}

						{activeTab === 'events' ? (
							<div className="space-y-4">
								<div className="flex justify-end">
									<Button
										type="button"
										variant="secondary"
										size="sm"
										onClick={() => setEventFormOpen((current) => !current)}
									>
										<CalendarPlus className="mr-2 h-4 w-4" />
										{eventFormOpen ? 'Close' : 'New event'}
									</Button>
								</div>

								{eventFormOpen ? (
									<form className="space-y-4 rounded-[24px] border border-indigo-100 bg-indigo-50/60 p-4" onSubmit={handleCreateEvent}>
										<Input
											label="Event title"
											value={eventTitle}
											onChange={(event) => setEventTitle(event.target.value)}
											placeholder="Event title"
										/>
										<div className="space-y-1.5">
											<label className="ml-1 text-sm font-medium text-gray-700">Description</label>
											<textarea
												value={eventDescription}
												onChange={(event) => setEventDescription(event.target.value)}
												rows={3}
												placeholder="Description"
												className="w-full rounded-bento border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-primary"
											/>
										</div>
										<div className="space-y-1.5">
											<label className="ml-1 text-sm font-medium text-gray-700">Date and time</label>
											<input
												type="datetime-local"
												value={eventDate}
												onChange={(event) => setEventDate(event.target.value)}
												className="w-full rounded-bento border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 focus:outline-none focus:ring-2 focus:ring-primary"
											/>
										</div>
										{eventError ? <p className="text-sm text-rose-600">{eventError}</p> : null}
										<Button type="submit" disabled={eventSubmitting}>
											{eventSubmitting ? <Loader className="mr-2 h-4 w-4 animate-spin" /> : <CalendarPlus className="mr-2 h-4 w-4" />}
											Create event
										</Button>
									</form>
								) : null}

								<div className="space-y-3">
									{workspace.events.length > 0 ? (
										workspace.events.map((event) => (
											<div key={event.id} className="rounded-[22px] border border-gray-100 bg-gray-50/80 p-4">
												<p className="font-semibold text-gray-900">{event.title}</p>
												{event.description ? <p className="mt-1 text-sm leading-6 text-gray-600">{event.description}</p> : null}
												<div className="mt-3 flex flex-wrap items-center gap-2 text-xs text-gray-500">
													<MetaPill label={formatDateTime(event.event_date)} />
												</div>
												<div className="mt-4 flex gap-2">
													<Button
														type="button"
														size="sm"
														variant={eventResponses[event.id] === 'going' ? 'primary' : 'secondary'}
														onClick={() => void handleEventResponse(event.id, 'going')}
													>
														Going
													</Button>
													<Button
														type="button"
														size="sm"
														variant={eventResponses[event.id] === 'not_going' ? 'danger' : 'ghost'}
														onClick={() => void handleEventResponse(event.id, 'not_going')}
													>
														Not going
													</Button>
												</div>
											</div>
										))
									) : (
										<EmptyState title="No events yet" />
									)}
								</div>
							</div>
						) : null}

						{activeTab === 'members' ? (
							membershipStatus !== 'accepted' ? (
								<div className="rounded-[22px] border border-dashed border-gray-200 bg-white px-4 py-8 text-center">
									<p className="font-semibold text-gray-900">Join to view members</p>
								</div>
							) : membersLoading ? (
								<div className="flex items-center justify-center py-12">
									<Loader className="h-6 w-6 animate-spin text-indigo-600" />
								</div>
							) : membersError ? (
								<UnavailableState description={membersError} />
							) : members.length > 0 ? (
								<div className="space-y-3">
									{members.map((member) => (
										<div key={member.id} className="flex items-center gap-3 rounded-[20px] border border-gray-100 bg-gray-50/80 p-3">
											{renderAvatar(member)}
											<div className="min-w-0">
												<p className="truncate text-sm font-semibold text-gray-900">
													{member.first_name} {member.last_name}
												</p>
												<p className="truncate text-xs text-gray-500">
													{member.nickname || member.id}
												</p>
											</div>
										</div>
									))}
								</div>
							) : (
								<EmptyState title="No members yet" />
							)
						) : null}

						{activeTab === 'manage' ? (
							<div className="space-y-4">
								{canInvite ? (
									<div className="space-y-3 rounded-[22px] border border-gray-100 bg-gray-50/80 p-4">
										<Input
											label="Invite people"
											value={inviteQuery}
											onChange={(event) => setInviteQuery(event.target.value)}
											placeholder="Search by name, nickname, or user ID"
										/>
										{inviteError ? <p className="text-sm text-rose-600">{inviteError}</p> : null}
										{inviteQuery.trim().length < 2 ? (
											<p className="text-sm text-gray-500">Type at least 2 characters to search.</p>
										) : inviteSearchLoading ? (
											<div className="flex items-center gap-2 text-sm text-gray-500">
												<Loader className="h-4 w-4 animate-spin text-indigo-600" />
												Searching people...
											</div>
										) : inviteResults.length > 0 ? (
											<div className="space-y-2">
												{inviteResults.map((user) => (
													<div key={user.id} className="flex items-center justify-between gap-3 rounded-[18px] border border-white bg-white px-3 py-3">
														<div className="flex min-w-0 items-center gap-3">
															{renderAvatar(user)}
															<div className="min-w-0">
																<p className="truncate text-sm font-semibold text-gray-900">
																	{user.first_name} {user.last_name}
																</p>
																<p className="truncate text-xs text-gray-500">
																	{user.nickname || user.id}
																</p>
															</div>
														</div>
														<Button
															type="button"
															size="sm"
															disabled={inviteSubmitting}
															onClick={() => void handleInviteUser(user)}
														>
															{inviteSubmitting ? <Loader className="mr-2 h-4 w-4 animate-spin" /> : <UserPlus className="mr-2 h-4 w-4" />}
															Invite
														</Button>
													</div>
												))}
											</div>
										) : (
											<p className="text-sm text-gray-500">No matching people available to invite.</p>
										)}
									</div>
								) : null}

								{canModerateRequests ? (
									requestsLoading ? (
										<div className="flex items-center justify-center py-12">
											<Loader className="h-6 w-6 animate-spin text-indigo-600" />
										</div>
									) : requestsError ? (
										<UnavailableState description={requestsError} />
									) : requests.length > 0 ? (
										<div className="space-y-3">
											{requests.map((user) => (
												<div key={user.id} className="rounded-[22px] border border-gray-100 bg-gray-50/80 p-4">
													<div className="flex items-center gap-3">
														{renderAvatar(user)}
														<div className="min-w-0">
															<p className="truncate text-sm font-semibold text-gray-900">
																{user.first_name} {user.last_name}
															</p>
															<p className="truncate text-xs text-gray-500">
																{user.nickname || user.id}
															</p>
														</div>
													</div>
													<div className="mt-4 flex gap-2">
														<Button type="button" size="sm" onClick={() => void handleJoinRequestDecision(user.id, true)}>
															Accept
														</Button>
														<Button type="button" size="sm" variant="secondary" onClick={() => void handleJoinRequestDecision(user.id, false)}>
															Decline
														</Button>
													</div>
												</div>
											))}
										</div>
									) : (
										<EmptyState title="No requests" />
									)
								) : !canInvite ? (
									<div className="rounded-[22px] border border-dashed border-gray-200 bg-white px-4 py-8 text-center">
										<p className="font-semibold text-gray-900">Nothing to manage</p>
									</div>
								) : null}
							</div>
						) : null}
					</div>
				</section>
			</div>
		</div>
	);
}

function MetaPill({ label }: { label: string }) {
	return (
		<span className="inline-flex items-center rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-600">
			{label}
		</span>
	);
}

function EmptyState({ title }: { title: string }) {
	return (
		<div className="rounded-[22px] border border-dashed border-gray-200 bg-white px-4 py-8 text-center">
			<p className="font-semibold text-gray-900">{title}</p>
		</div>
	);
}

function UnavailableState({ description }: { description: string }) {
	return (
		<div className="rounded-[22px] border border-amber-200 bg-amber-50 px-4 py-4">
			<p className="text-sm leading-6 text-amber-800">{description}</p>
		</div>
	);
}

function TabButton({
	label,
	active,
	onClick,
}: {
	label: string;
	active: boolean;
	onClick: () => void;
}) {
	return (
		<button
			type="button"
			onClick={onClick}
			className={`rounded-xl px-4 py-3 text-sm font-semibold transition-colors ${
				active ? 'bg-indigo-50 text-indigo-700' : 'bg-gray-50 text-gray-600 hover:bg-gray-100'
			}`}
		>
			{label}
		</button>
	);
}

function GroupPostCard({ post }: { post: PostResponse }) {
	const postImageUrl = resolveAssetUrl(post.image_url);
	return (
		<div className="bg-white rounded-xl border border-gray-100 overflow-hidden">
			<div className="p-4 flex items-center justify-between">
				<div className="flex items-center gap-3">
					{renderAvatar(post.author)}
					<div>
						<p className="text-sm font-bold">
							{post.author.first_name} {post.author.last_name}
						</p>
						<p className="text-xs text-gray-500">{formatDateTime(post.created_at)}</p>
					</div>
				</div>
			</div>

			{post.content ? (
				<div className="px-4 pb-4">
					<p className="text-sm leading-relaxed text-gray-800 whitespace-pre-wrap">
						{post.content}
					</p>
				</div>
			) : null}

			{postImageUrl ? (
				<div className="px-4 pb-4">
					<Image
						src={postImageUrl}
						alt=""
						unoptimized
						width={1200}
						height={900}
						className="w-full max-h-[32rem] rounded-xl border border-gray-100 object-cover bg-gray-50"
					/>
				</div>
			) : null}
		</div>
	);
}

function renderAvatar(user?: ProfileResponse['user'] | PostResponse['author']) {
	const avatarUrl = resolveAssetUrl(user?.avatar_path);
	const initials = `${user?.first_name?.[0] ?? ''}${user?.last_name?.[0] ?? ''}`.trim() || 'Y';

	if (avatarUrl) {
		return (
			<Image
				src={avatarUrl}
				alt={user ? `${user.first_name} ${user.last_name}` : 'User avatar'}
				unoptimized
				width={40}
				height={40}
				className="w-10 h-10 rounded-full object-cover flex-shrink-0"
			/>
		);
	}

	return (
		<div className="w-10 h-10 rounded-full bg-indigo-100 text-indigo-700 text-sm font-semibold flex items-center justify-center flex-shrink-0">
			{initials}
		</div>
	);
}

function extractError(reason: unknown, fallback: string) {
	return reason instanceof Error ? reason.message : fallback;
}

function formatDateTime(value: string) {
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) {
		return value;
	}

	return new Intl.DateTimeFormat('en-US', {
		month: 'short',
		day: 'numeric',
		year: 'numeric',
		hour: 'numeric',
		minute: '2-digit',
	}).format(date);
}
