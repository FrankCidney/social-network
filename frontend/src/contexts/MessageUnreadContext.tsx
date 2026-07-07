"use client";

import * as React from "react";
import {
  chatAPI,
  isAuthenticationError,
  profileAPI,
  type ChatConversation,
  type ChatMessage,
} from "@/lib/api";
import { useWebSocket } from "@/contexts/WebSocketContext";

type MessageUnreadContextValue = {
  unreadCount: number;
  refresh: () => Promise<void>;
  markConversationRead: (userId: string) => Promise<void>;
};

const MessageUnreadContext = React.createContext<MessageUnreadContextValue | undefined>(
  undefined
);

type ConversationsResult = Awaited<ReturnType<typeof chatAPI.getConversations>>;

function unwrapConversations(data: ConversationsResult): ChatConversation[] {
  if (!data) return [];
  return Array.isArray(data) ? data : data.conversations;
}

function sumUnread(unreadByUser: Record<string, number>) {
  return Object.values(unreadByUser).reduce((total, count) => total + count, 0);
}

export function MessageUnreadProvider({ children }: { children: React.ReactNode }) {
  const { socket } = useWebSocket();
  const [currentUserId, setCurrentUserId] = React.useState<string | null>(null);
  const [unreadByUser, setUnreadByUser] = React.useState<Record<string, number>>({});

  React.useEffect(() => {
    profileAPI
      .getMyProfile()
      .then((profile) => setCurrentUserId(profile.user.id))
      .catch(() => {});
  }, []);

  const refresh = React.useCallback(async () => {
    try {
      const data = await chatAPI.getConversations();
      const nextUnread: Record<string, number> = {};

      for (const conversation of unwrapConversations(data)) {
        if (conversation.unread_count > 0) {
          nextUnread[conversation.user.id] = conversation.unread_count;
        }
      }

      setUnreadByUser(nextUnread);
    } catch (err) {
      if (!isAuthenticationError(err)) {
        // The badge is best-effort; page-level chat errors still live on the messages page.
        setUnreadByUser({});
      }
    }
  }, []);

  React.useEffect(() => {
    void refresh();
  }, [refresh]);

  React.useEffect(() => {
    if (!socket || !currentUserId) return;

    const handleMessage = (event: MessageEvent) => {
      let data: { type?: string; payload?: ChatMessage };
      try {
        data = JSON.parse(event.data);
      } catch {
        return;
      }

      if (data.type !== "chat_message" || !data.payload) return;

      const incoming = data.payload;
      if (incoming.receiver_id !== currentUserId || incoming.sender_id === currentUserId) {
        return;
      }

      setUnreadByUser((current) => ({
        ...current,
        [incoming.sender_id]: (current[incoming.sender_id] ?? 0) + 1,
      }));
    };

    socket.addEventListener("message", handleMessage);
    return () => socket.removeEventListener("message", handleMessage);
  }, [currentUserId, socket]);

  const markConversationRead = React.useCallback(async (userId: string) => {
    setUnreadByUser((current) => {
      if (!current[userId]) return current;

      const next = { ...current };
      delete next[userId];
      return next;
    });

    await chatAPI.markConversationRead(userId);
  }, []);

  const value = React.useMemo<MessageUnreadContextValue>(
    () => ({
      unreadCount: sumUnread(unreadByUser),
      refresh,
      markConversationRead,
    }),
    [markConversationRead, refresh, unreadByUser]
  );

  return (
    <MessageUnreadContext.Provider value={value}>
      {children}
    </MessageUnreadContext.Provider>
  );
}

export function useMessageUnread() {
  const context = React.useContext(MessageUnreadContext);
  if (!context) {
    throw new Error("useMessageUnread must be used within MessageUnreadProvider");
  }
  return context;
}
