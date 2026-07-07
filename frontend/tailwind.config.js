/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        canvas: "#F9FAFB",
        surface: "#FFFFFF",
        primary: "#4f46e5",
        "text-main": "#111827",
        privacy: {
          public: "#10B981",
          private: "#EF4444",
        },
      },
      borderRadius: {
        bento: "12px",
      }
    },
  },
  plugins: [],
}
