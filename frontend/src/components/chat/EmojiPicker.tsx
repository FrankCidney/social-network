"use client";

import * as React from "react";
import { Smile } from "lucide-react";

const EMOJI_GROUPS = [
  ["😀", "😄", "😂", "😊", "😍", "😘", "😎", "🥳"],
  ["👍", "👎", "👏", "🙌", "🙏", "💪", "🤝", "✌️"],
  ["❤️", "💙", "💚", "💛", "💜", "🔥", "✨", "🎉"],
  ["👋", "🤔", "😭", "😅", "😮", "😴", "😡", "😇"],
];

type EmojiPickerProps = {
  onSelect: (emoji: string) => void;
  disabled?: boolean;
};

export function EmojiPicker({ onSelect, disabled }: EmojiPickerProps) {
  const [open, setOpen] = React.useState(false);
  const rootRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    if (!open) {
      return;
    }

    const handlePointerDown = (event: MouseEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setOpen(false);
      }
    };

    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open]);

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        disabled={disabled}
        title="Add emoji"
        aria-label="Add emoji"
        aria-expanded={open}
        onClick={() => setOpen((current) => !current)}
        className="flex h-9 w-9 items-center justify-center rounded-full text-gray-500 transition-colors hover:bg-gray-100 hover:text-indigo-600 focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:pointer-events-none disabled:opacity-40"
      >
        <Smile className="h-4 w-4" />
      </button>

      {open ? (
        <div className="absolute bottom-full right-0 z-30 mb-2 w-64 rounded-xl border border-gray-200 bg-white p-3 shadow-lg">
          <div className="grid grid-cols-8 gap-1">
            {EMOJI_GROUPS.flat().map((emoji) => (
              <button
                key={emoji}
                type="button"
                title={emoji}
                aria-label={`Insert ${emoji}`}
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => onSelect(emoji)}
                className="flex h-8 w-8 items-center justify-center rounded-lg text-lg leading-none transition-colors hover:bg-indigo-50 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                {emoji}
              </button>
            ))}
          </div>
        </div>
      ) : null}
    </div>
  );
}
