"use client";

import Link from "next/link";
import React, { useState, useEffect, useRef, useMemo, useCallback } from "react";
import { useSearchParams } from "next/navigation";
import { Send, Search, MessageSquare } from "lucide-react";
import { EmojiPicker } from "@/components/chat/EmojiPicker";
import { useMessageUnread } from "@/contexts/MessageUnreadContext";
import { useWebSocket } from "@/contexts/WebSocketContext";
import {
  chatAPI,
  isAuthenticationError,
  isForbiddenError,
  profileAPI,
  resolveAssetUrl,
  ChatConversation,
  ChatMessage,
  PublicUser,
} from "@/lib/api";

function displayName(user: PublicUser) {
  return user.nickname || `${user.first_name} ${user.last_name}`;
}

function initials(user: PublicUser) {
  return `${user.first_name?.[0] ?? ""}${user.last_name?.[0] ?? ""}`.toUpperCase();
}

function formatTime(iso?: string) {
  if (!iso) return "";
  return new Date(iso).toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
}

// The conversations/messages endpoints may return either a bare array or a
// wrapped object depending on backend version — normalize both shapes.
function unwrapConversations(data: ConversationsResult): ChatConversation[] {
  if (!data) return [];
  return Array.isArray(data) ? data : data.conversations;
}
function unwrapMessages(data: MessagesResult): ChatMessage[] {
  if (!data) return [];
  return Array.isArray(data) ? data : data.messages;
}

function isOptimisticMessage(message: ChatMessage) {
  return message.id.startsWith("temp-");
}
type ConversationsResult = Awaited<ReturnType<typeof chatAPI.getConversations>>;
type MessagesResult = Awaited<ReturnType<typeof chatAPI.getMessages>>;

export default function MessagesPage() {
  const { socket } = useWebSocket();
  const { markConversationRead } = useMessageUnread();
  const searchParams = useSearchParams();
  const requestedUserId = searchParams.get("user");

  const [currentUser, setCurrentUser] = useState<PublicUser | null>(null);

  const [conversations, setConversations] = useState<ChatConversation[]>([]);
  const [conversationsLoading, setConversationsLoading] = useState(true);
  const [conversationsError, setConversationsError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [requestedUser, setRequestedUser] = useState<PublicUser | null>(null);
  const [requestedUserLoading, setRequestedUserLoading] = useState(false);
  const [requestedUserError, setRequestedUserError] = useState<string | null>(null);

  const [selectedUserId, setSelectedUserId] = useState<string | null>(null);

  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [messagesLoading, setMessagesLoading] = useState(false);
  const [messagesError, setMessagesError] = useState<string | null>(null);

  const [newMessage, setNewMessage] = useState("");
  const [sending, setSending] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messageInputRef = useRef<HTMLInputElement>(null);

  // Keep the selected user id in a ref so the WebSocket handler (registered
  // once) always sees the latest selection without re-subscribing.
  const selectedUserIdRef = useRef<string | null>(null);
  useEffect(() => {
    selectedUserIdRef.current = selectedUserId;
  }, [selectedUserId]);

  // Load current user, once.
  useEffect(() => {
    profileAPI
      .getMyProfile()
      .then((profile) => setCurrentUser(profile.user))
      .catch(() => {});
  }, []);

  const loadConversations = useCallback(async () => {
    try {
      setConversationsLoading(true);
      const data = await chatAPI.getConversations();
      const list = unwrapConversations(data);
      setConversations(list);
      setConversationsError(null);
      setSelectedUserId((prev) => prev ?? list[0]?.user.id ?? null);
    } catch (err) {
      if (!isAuthenticationError(err)) {
        setConversationsError(err instanceof Error ? err.message : "Could not load your conversations.");
      }
    } finally {
      setConversationsLoading(false);
    }
  }, []);

  useEffect(() => {
    loadConversations();
  }, [loadConversations]);

  useEffect(() => {
    if (!requestedUserId || !currentUser || requestedUserId === currentUser.id) {
      setRequestedUser(null);
      setRequestedUserError(null);
      return;
    }

    let cancelled = false;

    const loadRequestedUser = async () => {
      setRequestedUserLoading(true);
      setRequestedUserError(null);
      try {
        const profile = await profileAPI.getProfile(requestedUserId);
        if (!cancelled) {
          setRequestedUser(profile.user);
          setSelectedUserId(requestedUserId);
        }
      } catch (err) {
        if (!cancelled) {
          setRequestedUser(null);
          setRequestedUserError(
            err instanceof Error ? err.message : "Could not open this conversation."
          );
        }
      } finally {
        if (!cancelled) {
          setRequestedUserLoading(false);
        }
      }
    };

    void loadRequestedUser();

    return () => {
      cancelled = true;
    };
  }, [currentUser, requestedUserId]);

  const requestedConversation = useMemo<ChatConversation | null>(() => {
    if (!requestedUserId || !requestedUser || conversations.some((conversation) => conversation.user.id === requestedUserId)) {
      return null;
    }

    return {
      user: requestedUser,
      unread_count: 0,
    };
  }, [conversations, requestedUser, requestedUserId]);

  const allConversations = useMemo(
    () => (requestedConversation ? [requestedConversation, ...conversations] : conversations),
    [conversations, requestedConversation]
  );

  const loadMessages = useCallback(async (userId: string) => {
    try {
      setMessagesLoading(true);
      const data = await chatAPI.getMessages(userId);
      setMessages(unwrapMessages(data));
      setMessagesError(null);
      markConversationRead(userId).catch(() => {});
      setConversations((prev) =>
        prev.map((c) => (c.user.id === userId ? { ...c, unread_count: 0 } : c))
      );
    } catch (err) {
      if (!isAuthenticationError(err)) {
        setMessagesError(err instanceof Error ? err.message : "Could not load this conversation.");
      }
    } finally {
      setMessagesLoading(false);
    }
  }, [markConversationRead]);

  useEffect(() => {
    if (selectedUserId) {
      loadMessages(selectedUserId);
    } else {
      setMessages([]);
    }
  }, [selectedUserId, loadMessages]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  // Live incoming messages over the shared WebSocket connection.
  useEffect(() => {
    if (!socket) return;

    const handleMessage = (event: MessageEvent) => {
      let data: { type?: string; payload?: ChatMessage };
      try {
        data = JSON.parse(event.data);
      } catch (err) {
        console.error("Failed to parse chat message", err);
        return;
      }
      if (data.type !== "chat_message" || !data.payload) return;

      const incoming = data.payload;
      const openUserId = selectedUserIdRef.current;
      const isForOpenConversation =
        incoming.sender_id === openUserId || incoming.receiver_id === openUserId;

      if (isForOpenConversation) {
        setMessages((prev) => {
          if (prev.some((message) => message.id === incoming.id)) {
            return prev;
          }

          const optimisticIndex = prev.findIndex(
            (message) =>
              isOptimisticMessage(message) &&
              message.sender_id === incoming.sender_id &&
              message.receiver_id === incoming.receiver_id &&
              message.content === incoming.content
          );

          if (optimisticIndex === -1) {
            return [...prev, incoming];
          }

          return prev.map((message, index) => (index === optimisticIndex ? incoming : message));
        });
        markConversationRead(incoming.sender_id).catch(() => {});
      }

      // Bump the relevant conversation's preview/unread badge regardless of
      // whether it's currently open.
      setConversations((prev) => {
        const otherUserId =
          incoming.sender_id === currentUser?.id ? incoming.receiver_id : incoming.sender_id;
        const exists = prev.some((c) => c.user.id === otherUserId);
        if (!exists) {
          // New conversation we don't have yet — refetch the list.
          loadConversations();
          return prev;
        }
        return prev.map((c) =>
          c.user.id === otherUserId
            ? {
                ...c,
                last_message: incoming,
                unread_count: isForOpenConversation ? 0 : c.unread_count + 1,
              }
            : c
        );
      });
    };

    socket.addEventListener("message", handleMessage);
    return () => socket.removeEventListener("message", handleMessage);
  }, [socket, currentUser?.id, loadConversations, markConversationRead]);

  const filteredConversations = useMemo(
    () =>
      allConversations.filter((c) =>
        displayName(c.user).toLowerCase().includes(searchQuery.toLowerCase())
      ),
    [allConversations, searchQuery]
  );

  const activeConversation = allConversations.find((c) => c.user.id === selectedUserId) ?? null;

  const handleSendMessage = async (e: React.FormEvent) => {
    e.preventDefault();
    const content = newMessage.trim();
    if (!content || !selectedUserId || sending || !currentUser) return;

    const optimistic: ChatMessage = {
      id: `temp-${Date.now()}`,
      sender_id: currentUser.id,
      receiver_id: selectedUserId,
      content,
      created_at: new Date().toISOString(),
    };
    setMessages((prev) => [...prev, optimistic]);
    setNewMessage("");

    try {
      setSending(true);
      const saved = await chatAPI.sendMessage({ receiver_id: selectedUserId, content });
      setMessages((prev) => {
        const replaced = prev.map((message) => (message.id === optimistic.id ? saved : message));
        return replaced.filter(
          (message, index, current) => current.findIndex((candidate) => candidate.id === message.id) === index
        );
      });
      setConversations((prev) =>
        prev.some((c) => c.user.id === selectedUserId)
          ? prev.map((c) =>
              c.user.id === selectedUserId ? { ...c, last_message: saved } : c
            )
          : activeConversation
            ? [{ ...activeConversation, last_message: saved, unread_count: 0 }, ...prev]
            : prev
      );
      setMessagesError(null);
    } catch (err) {
      setMessages((prev) => prev.filter((m) => m.id !== optimistic.id));
      setMessagesError(err instanceof Error ? err.message : "Message failed to send.");
      setNewMessage(content);
    } finally {
      setSending(false);
    }
  };

  const insertEmoji = (emoji: string) => {
    const input = messageInputRef.current;
    const start = input?.selectionStart ?? newMessage.length;
    const end = input?.selectionEnd ?? newMessage.length;
    const nextMessage = `${newMessage.slice(0, start)}${emoji}${newMessage.slice(end)}`;

    setNewMessage(nextMessage);

    requestAnimationFrame(() => {
      input?.focus();
      const nextCursor = start + emoji.length;
      input?.setSelectionRange(nextCursor, nextCursor);
    });
  };

  return (
    <div className="bg-white rounded-xl border border-gray-100 shadow-sm overflow-hidden flex h-[calc(100vh-140px)]">
      {/* Sidebar: Chats List */}
      <div className="w-1/3 border-r border-gray-100 flex flex-col bg-gray-50">
        <div className="p-4 border-b border-gray-100">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input
              type="text"
              placeholder="Search messages..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full bg-white border border-gray-200 rounded-full py-2 pl-10 pr-4 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>
        </div>
        <div className="overflow-y-auto flex-1 p-2 space-y-1">
          {conversationsLoading &&
            [1, 2, 3].map((i) => (
              <div key={i} className="flex items-center gap-3 p-3 animate-pulse">
                <div className="w-12 h-12 rounded-full bg-gray-200 shrink-0" />
                <div className="flex-1 space-y-2">
                  <div className="w-24 h-3 bg-gray-200 rounded" />
                  <div className="w-32 h-2 bg-gray-200 rounded" />
                </div>
              </div>
            ))}

          {!conversationsLoading && conversationsError && (
            <div className="p-4 text-center">
              <p className="text-sm text-red-600 mb-2">{conversationsError}</p>
              <button
                onClick={loadConversations}
                className="text-sm text-indigo-600 hover:underline"
              >
                Retry
              </button>
            </div>
          )}

          {!conversationsLoading &&
            !conversationsError &&
            filteredConversations.map((conv) => {
              const avatarUrl = resolveAssetUrl(conv.user.avatar_path);
              return (
                <div
                  key={conv.user.id}
                  onClick={() => setSelectedUserId(conv.user.id)}
                  className={`flex items-center gap-3 p-3 rounded-lg hover:bg-white cursor-pointer transition-colors ${
                    selectedUserId === conv.user.id ? "bg-white shadow-sm" : ""
                  }`}
                >
                  {avatarUrl ? (
                    <img
                      src={avatarUrl}
                      alt={displayName(conv.user)}
                      className="w-12 h-12 rounded-full object-cover shrink-0"
                    />
                  ) : (
                    <div className="w-12 h-12 rounded-full bg-indigo-100 shrink-0 flex items-center justify-center text-indigo-600 font-semibold text-sm">
                      {initials(conv.user)}
                    </div>
                  )}
                  <div className="flex-1 min-w-0">
                    <div className="flex justify-between items-baseline mb-1">
                      <h3 className="font-semibold text-gray-900 truncate">
                        {displayName(conv.user)}
                      </h3>
                      <span className="text-xs text-gray-400">
                        {formatTime(conv.last_message?.created_at)}
                      </span>
                    </div>
                    <div className="flex justify-between items-center gap-2">
                      <p className="text-sm text-gray-500 truncate">
                        {conv.last_message?.content ?? "No messages yet"}
                      </p>
                      {conv.unread_count > 0 && (
                        <span className="bg-indigo-600 text-white text-xs font-bold w-5 h-5 rounded-full flex items-center justify-center shrink-0">
                          {conv.unread_count}
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}

          {!conversationsLoading && !conversationsError && filteredConversations.length === 0 && (
            <div className="p-6 text-center">
              <MessageSquare className="w-8 h-8 text-gray-300 mx-auto mb-3" />
              <p className="text-sm text-gray-500">
                {requestedUserLoading
                  ? "Opening conversation..."
                  : requestedUserError || "No conversations found"}
              </p>
            </div>
          )}
        </div>
      </div>

      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col bg-white">
        {activeConversation ? (
          <>
            {/* Chat Header */}
            <div className="h-16 border-b border-gray-100 flex items-center px-6 justify-between">
              <div className="flex items-center gap-3">
                <Link href={`/profile/${activeConversation.user.id}`} className="contents">
                  {resolveAssetUrl(activeConversation.user.avatar_path) ? (
                    <img
                      src={resolveAssetUrl(activeConversation.user.avatar_path)}
                      alt={displayName(activeConversation.user)}
                      className="w-10 h-10 rounded-full object-cover"
                    />
                  ) : (
                    <div className="w-10 h-10 rounded-full bg-indigo-100 flex items-center justify-center text-indigo-600 font-semibold text-sm">
                      {initials(activeConversation.user)}
                    </div>
                  )}
                  <div>
                    <h2 className="font-bold text-gray-900 hover:text-indigo-600">{displayName(activeConversation.user)}</h2>
                  </div>
                </Link>
              </div>
            </div>

            {/* Chat Messages */}
            <div className="flex-1 overflow-y-auto p-6 space-y-4">
              {messagesLoading && (
                <div className="space-y-4">
                  {[1, 2, 3].map((i) => (
                    <div key={i} className={`flex ${i % 2 === 0 ? "justify-end" : "justify-start"}`}>
                      <div className="w-40 h-9 bg-gray-100 rounded-2xl animate-pulse" />
                    </div>
                  ))}
                </div>
              )}

              {!messagesLoading && messagesError && (
                <div className="text-center text-sm text-red-600 py-4">{messagesError}</div>
              )}

              {!messagesLoading && !messagesError && messages.length === 0 && (
                <div className="text-center text-xs text-gray-400 py-8">
                  No messages yet. Say hello!
                </div>
              )}

              {!messagesLoading &&
                messages.map((m) => {
                  const isOwn = m.sender_id === currentUser?.id;
                  return (
                    <div key={m.id} className={`flex ${isOwn ? "justify-end" : "justify-start"}`}>
                      <div
                        className={
                          isOwn
                            ? "bg-indigo-600 text-white rounded-2xl rounded-tr-sm px-4 py-2 max-w-[70%] shadow-sm"
                            : "bg-gray-100 text-gray-800 rounded-2xl rounded-tl-sm px-4 py-2 max-w-[70%]"
                        }
                      >
                        {m.content}
                      </div>
                    </div>
                  );
                })}
              <div ref={messagesEndRef} />
            </div>

            {/* Message Input */}
            <div className="p-4 bg-white border-t border-gray-100">
              <form onSubmit={handleSendMessage} className="flex gap-2">
                <div className="relative flex-1">
                  <input
                    ref={messageInputRef}
                    type="text"
                    value={newMessage}
                    onChange={(e) => setNewMessage(e.target.value)}
                    placeholder="Type a message..."
                    disabled={sending}
                    className="w-full bg-gray-50 border border-gray-200 rounded-full py-2 pl-4 pr-12 focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:opacity-60"
                  />
                  <div className="absolute right-1 top-1/2 -translate-y-1/2">
                    <EmojiPicker onSelect={insertEmoji} disabled={sending} />
                  </div>
                </div>
                <button
                  type="submit"
                  disabled={!newMessage.trim() || sending}
                  className="bg-indigo-600 text-white p-2 w-10 h-10 rounded-full flex items-center justify-center hover:bg-indigo-700 transition-colors disabled:opacity-50"
                >
                  <Send className="w-4 h-4 ml-1" />
                </button>
              </form>
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <div className="text-center">
              <MessageSquare className="w-12 h-12 text-gray-300 mx-auto mb-4" />
              <h3 className="text-lg font-bold text-gray-900 mb-2">
                {conversationsLoading ? "Loading conversations…" : "Select a conversation"}
              </h3>
              <p className="text-gray-500 text-sm">
                {conversationsLoading
                  ? "Hang tight while we pull in your chats."
                  : "Choose a chat from the sidebar to start messaging"}
              </p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
