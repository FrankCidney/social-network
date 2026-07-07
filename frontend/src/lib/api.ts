const API_BASE_URL = (process.env.NEXT_PUBLIC_API_URL ?? '').replace(/\/$/, '');

type RequestOptions = Omit<RequestInit, 'body'> & {
  body?: BodyInit | Record<string, unknown> | unknown[];
};

export class ApiError extends Error {
	status: number;

	constructor(message: string, status: number) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
	}
}

export function isAuthenticationError(error: unknown) {
	return error instanceof ApiError && error.status === 401;
}

export function isForbiddenError(error: unknown) {
	return error instanceof ApiError && error.status === 403;
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  const isFormData = options.body instanceof FormData;
  let body: BodyInit | undefined;

  if (isFormData) {
    body = options.body as FormData;
  } else if (options.body !== undefined) {
    body = JSON.stringify(options.body);
  }

  if (options.body !== undefined && !isFormData && !headers.has('content-type')) {
    headers.set('content-type', 'application/json');
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers,
    credentials: 'include',
    body,
  });

  if (!response.ok) {
    throw new ApiError(await getErrorMessage(response), response.status);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}

async function getErrorMessage(response: Response) {
	try {
		const data = await response.json();

		if (typeof data?.error?.message === 'string') {
			return data.error.message;
		}

		if (typeof data?.error === 'string') {
			return data.error;
		}

		if (typeof data?.message === 'string') {
			return data.message;
		}
	} catch {
		// Fall back to the status text below when the response is not JSON.
	}

	return response.statusText || 'Request failed';
}

export type RegisterPayload = {
	email: string;
	password: string;
	first_name: string;
	last_name: string;
	dob: string;
	nickname?: string;
	about_me?: string;
};

export type LoginPayload = {
	email: string;
	password: string;
};

export type AuthResponse = {
	user: PublicUser;
	token: string;
};

export type PublicUser = {
	id: string;
	email?: string;
	first_name: string;
	last_name: string;
	nickname?: string;
	avatar_path?: string;
	is_public: boolean;
};

export type PostResponse = {
	id: string;
	author: PublicUser;
	group_id?: string;
	content?: string;
	image_url?: string;
	privacy: string;
	created_at: string;
};

export type PostListResponse = {
	posts: PostResponse[];
	total: number;
	limit: number;
	offset: number;
};

export type ProfileResponse = {
	user: PublicUser;
	email?: string;
	about_me?: string;
	dob?: string;
	follower_count: number;
	following_count: number;
	post_count: number;
	can_view_full_profile?: boolean;
	is_own_profile?: boolean;
	is_following?: boolean;
	follow_request_status?: 'pending' | 'accepted' | 'declined';
};

export type UpdateProfilePayload = {
	first_name?: string;
	last_name?: string;
	dob?: string;
	nickname?: string;
	about_me?: string;
	is_public?: boolean;
};

export type UploadAvatarResponse = {
	avatar_path: string;
};

export type FollowListResponse = {
	users: PublicUser[];
	total: number;
	limit: number;
	offset: number;
};

export type UserSearchResult = PublicUser & {
	is_following: boolean;
	follow_request_status?: 'pending' | 'accepted' | 'declined';
};

export type UserSearchParams = {
	limit?: number;
	exclude_group_id?: string;
};

export type CommentResponse = {
	id: string;
	author: PublicUser;
	content?: string;
	image_url?: string;
	depth: number;
	created_at: string;
	replies: CommentResponse[];
};

export type CreatePostPayload = {
	content?: string;
	privacy: string;
	visible_to?: string[];
	group_id?: string;
};

export type CreateCommentPayload = {
	content?: string;
	parent_comment_id?: string;
};

export type CreatedCommentResponse = {
	id: string;
	post_id: string;
	user_id: string;
	content?: string;
	image_url?: string;
	parent_comment_id?: string;
	created_at: string;
};

export type UploadPostImageResponse = {
	image_url: string;
};

export type UploadCommentImageResponse = {
	image_url: string;
};

export type CreatedPostResponse = {
	id: string;
	group_id?: string;
	content?: string;
	image_url?: string;
	privacy: string;
	created_at: string;
};

export type Group = {
	id: string;
	creator_id: string;
	title: string;
	description: string;
	created_at: string;
};

export type GroupDetail = Group & {
	creator: PublicUser;
	is_creator: boolean;
	membership_status: 'none' | 'invited' | 'requested' | 'accepted' | 'declined';
};

export type CreateGroupPayload = {
	title: string;
	description: string;
};

export type GroupEvent = {
	id: string;
	group_id: string;
	creator_id: string;
	title: string;
	description: string;
	event_date: string;
	created_at: string;
};

export type CreateGroupEventPayload = {
	title: string;
	description: string;
	event_date: string;
};

export type InviteUserPayload = {
	invitee_id: string;
};

export type GroupMessage = {
	id: string;
	sender_id: string;
	group_id?: string;
	content: string;
	created_at: string;
};

export const authAPI = {
	register(payload: RegisterPayload) {
		return request<AuthResponse>('/api/auth/register', {
			method: 'POST',
			body: payload,
		});
	},

	login(payload: LoginPayload) {
		return request<AuthResponse>('/api/auth/login', {
			method: 'POST',
			body: payload,
		});
	},

	logout() {
		return request<void>('/api/auth/logout', {
			method: 'POST',
		});
	},
};

export const groupAPI = {
	getGroups() {
		return request<Group[] | { groups?: Group[] } | null>('/api/groups');
	},

	getGroup(groupId: string) {
		return request<GroupDetail>(`/api/groups/${groupId}`);
	},

	createGroup(payload: CreateGroupPayload) {
		return request<Group>('/api/groups', {
			method: 'POST',
			body: payload,
		});
	},

	requestJoin(groupId: string) {
		return request<{ message: string }>(`/api/groups/${groupId}/join`, {
			method: 'POST',
		});
	},

	getEvents(groupId: string) {
		return request<GroupEvent[] | null>(`/api/groups/${groupId}/events`);
	},

	createEvent(groupId: string, payload: CreateGroupEventPayload) {
		return request<GroupEvent>(`/api/groups/${groupId}/events`, {
			method: 'POST',
			body: payload,
		});
	},

	getMembers(groupId: string) {
		return request<PublicUser[] | null>(`/api/groups/${groupId}/members`);
	},

	inviteUser(groupId: string, payload: InviteUserPayload) {
		return request<{ message: string }>(`/api/groups/${groupId}/invite`, {
			method: 'POST',
			body: payload,
		});
	},

	acceptInvite(groupId: string) {
		return request<void>(`/api/groups/${groupId}/invite/accept`, {
			method: 'POST',
		});
	},

	declineInvite(groupId: string) {
		return request<void>(`/api/groups/${groupId}/invite/decline`, {
			method: 'POST',
		});
	},

	getJoinRequests(groupId: string) {
		return request<PublicUser[] | null>(`/api/groups/${groupId}/requests`);
	},

	acceptJoinRequest(groupId: string, userId: string) {
		return request<void>(`/api/groups/${groupId}/requests/${userId}/accept`, {
			method: 'POST',
		});
	},

	declineJoinRequest(groupId: string, userId: string) {
		return request<void>(`/api/groups/${groupId}/requests/${userId}/decline`, {
			method: 'POST',
		});
	},

	rsvpEvent(eventId: string, status: 'going' | 'not_going') {
		return request<void>(`/api/events/${eventId}/rsvp`, {
			method: 'POST',
			body: { status },
		});
	},
};

export const profileAPI = {
	getMyProfile() {
		return request<ProfileResponse>('/api/profile');
	},

	getProfile(userId: string) {
		return request<ProfileResponse>(`/api/profile/${userId}`);
	},

	async updateProfile(payload: UpdateProfilePayload) {
		await request<void>('/api/profile', {
			method: 'PUT',
			body: payload,
		});

		return request<ProfileResponse>('/api/profile');
	},

	uploadAvatar(file: File) {
		const formData = new FormData();
		formData.append('avatar', file);

		return request<UploadAvatarResponse>('/api/profile/avatar', {
			method: 'POST',
			body: formData,
		});
	},

	getFollowers(userId: string, limit?: number, offset?: number) {
		const params = new URLSearchParams();
		if (limit !== undefined) params.set('limit', String(limit));
		if (offset !== undefined) params.set('offset', String(offset));
		const query = params.toString();
		return request<FollowListResponse>(`/api/users/${userId}/followers${query ? `?${query}` : ''}`);
	},

	getFollowing(userId: string, limit?: number, offset?: number) {
		const params = new URLSearchParams();
		if (limit !== undefined) params.set('limit', String(limit));
		if (offset !== undefined) params.set('offset', String(offset));
		const query = params.toString();
		return request<FollowListResponse>(`/api/users/${userId}/following${query ? `?${query}` : ''}`);
	},

	searchUsers(searchQuery: string, params: UserSearchParams = {}) {
		const query = searchQuery.trim();
		if (!query) {
			return Promise.resolve([] as UserSearchResult[]);
		}

		const urlParams = new URLSearchParams();
		urlParams.set('q', query);
		if (params.limit !== undefined) urlParams.set('limit', String(params.limit));
		if (params.exclude_group_id) urlParams.set('exclude_group_id', params.exclude_group_id);

		return request<UserSearchResult[]>(`/api/users/search?${urlParams.toString()}`);
	},
};

export const followAPI = {
	follow(userId: string) {
		return request<void>(`/api/follow/${userId}`, {
			method: 'POST',
		});
	},

	unfollow(userId: string) {
		return request<void>(`/api/follow/${userId}`, {
			method: 'DELETE',
		});
	},
};

export const feedAPI = {
	getFeed(limit?: number, offset?: number) {
		const params = new URLSearchParams();
		if (limit !== undefined) params.set('limit', String(limit));
		if (offset !== undefined) params.set('offset', String(offset));
		const query = params.toString();
		return request<PostListResponse>(`/api/feed${query ? `?${query}` : ''}`);
	},

	createPost(payload: CreatePostPayload) {
		return request<CreatedPostResponse>('/api/posts', {
			method: 'POST',
			body: payload,
		});
	},

	getPost(postId: string) {
		return request<PostResponse>(`/api/posts/${postId}`);
	},

	getUserPosts(userId: string, limit?: number, offset?: number) {
		const params = new URLSearchParams();
		if (limit !== undefined) params.set('limit', String(limit));
		if (offset !== undefined) params.set('offset', String(offset));
		const query = params.toString();
		return request<PostListResponse>(`/api/users/${userId}/posts${query ? `?${query}` : ''}`);
	},

	getGroupPosts(groupId: string, limit?: number, offset?: number) {
		const params = new URLSearchParams();
		if (limit !== undefined) params.set('limit', String(limit));
		if (offset !== undefined) params.set('offset', String(offset));
		const query = params.toString();
		return request<PostListResponse>(`/api/groups/${groupId}/posts${query ? `?${query}` : ''}`);
	},

	uploadPostImage(postId: string, file: File) {
		const formData = new FormData();
		formData.append('image', file);

		return request<UploadPostImageResponse>(`/api/posts/${postId}/image`, {
			method: 'POST',
			body: formData,
		});
	},

	getComments(postId: string) {
		return request<CommentResponse[]>(`/api/posts/${postId}/comments`);
	},

	createComment(postId: string, payload: CreateCommentPayload) {
		return request<CreatedCommentResponse>(`/api/posts/${postId}/comments`, {
			method: 'POST',
			body: payload,
		});
	},

	uploadCommentImage(commentId: string, file: File) {
		const formData = new FormData();
		formData.append('image', file);

		return request<UploadCommentImageResponse>(`/api/comments/${commentId}/image`, {
			method: 'POST',
			body: formData,
		});
	},
};

export function resolveAssetUrl(path?: string) {
	if (!path) {
		return undefined;
	}

	if (/^https?:\/\//i.test(path)) {
		return path;
	}

	const normalizedPath = path.startsWith('/') ? path : `/${path}`;
	return API_BASE_URL ? `${API_BASE_URL}${normalizedPath}` : normalizedPath;
}

// --- Chat / Messaging API -----------------------------------------------
// Conversations are keyed by the *other* user's id (1:1 chat), matching the
// shape your MessagesPage already assumed with `receiver_id`. Adjust the
// endpoint paths below if your backend routes differ.

export type ChatMessage = {
	id: string;
	sender_id: string;
	receiver_id: string;
	content: string;
	created_at: string;
	read_at?: string;
};

export type ChatConversation = {
	user: PublicUser;
	last_message?: ChatMessage;
	unread_count: number;
};

export type ConversationListResponse = {
	conversations: ChatConversation[];
};

export type ChatMessageListResponse = {
	messages: ChatMessage[];
	total: number;
	limit: number;
	offset: number;
};

export type SendMessagePayload = {
	receiver_id?: string;
	group_id?: string;
	content: string;
};

export const chatAPI = {
	// List all conversations for the current user, most recent first.
	getConversations() {
		return request<ConversationListResponse | ChatConversation[]>('/api/chat/conversations');
	},

	// Message history with a specific user.
	getMessages(userId: string, limit?: number, offset?: number) {
		const params = new URLSearchParams();
		if (limit !== undefined) params.set('limit', String(limit));
		if (offset !== undefined) params.set('offset', String(offset));
		const query = params.toString();
		return request<ChatMessageListResponse | ChatMessage[]>(
			`/api/chat/messages/${userId}${query ? `?${query}` : ''}`
		);
	},

	// Send a message; the backend broadcasts it over WebSocket to the receiver.
	sendMessage(payload: SendMessagePayload) {
		return request<ChatMessage>('/api/chat/messages', {
			method: 'POST',
			body: payload,
		});
	},

	getGroupMessages(groupId: string, limit?: number, offset?: number) {
		const params = new URLSearchParams();
		if (limit !== undefined) params.set('limit', String(limit));
		if (offset !== undefined) params.set('offset', String(offset));
		const query = params.toString();
		return request<GroupMessage[] | null>(`/api/groups/${groupId}/messages${query ? `?${query}` : ''}`);
	},

	// Mark all messages from a user as read.
	markConversationRead(userId: string) {
		return request<void>(`/api/chat/conversations/${userId}/read`, {
			method: 'POST',
		});
	},
};

// --- Notifications API ---------------------------------------------------

export type NotificationItem = {
	id: string;
	type: string;
	actor_id: string;
	actor?: PublicUser;
	group_id?: string;
	event_id?: string;
	message?: string;
	created_at: string;
	is_read: boolean;
	is_resolved?: boolean;
};

export type NotificationListResponse = {
	notifications: NotificationItem[];
	total: number;
	limit: number;
	offset: number;
};

export const notificationsAPI = {
	// List notifications for the current user, most recent first.
	getNotifications(limit?: number, offset?: number) {
		const params = new URLSearchParams();
		if (limit !== undefined) params.set('limit', String(limit));
		if (offset !== undefined) params.set('offset', String(offset));
		const query = params.toString();
		return request<NotificationListResponse | NotificationItem[]>(
			`/api/notifications${query ? `?${query}` : ''}`
		);
	},

	// Mark a single notification as read.
	markAsRead(notificationId: string) {
		return request<void>(`/api/notifications/${notificationId}/read`, {
			method: 'POST',
		});
	},

	resolve(notificationId: string) {
		return request<void>(`/api/notifications/${notificationId}/resolve`, {
			method: 'POST',
		});
	},

	// Mark every notification as read.
	markAllAsRead() {
		return request<void>('/api/notifications/read-all', {
			method: 'POST',
		});
	},

	// Accept/decline a follow request notification.
	respondToFollowRequest(actorId: string, accept: boolean) {
		return request<void>(`/api/follow/${actorId}/${accept ? 'accept' : 'decline'}`, {
			method: 'POST',
		});
	},
};
