/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./internal/web/**/*.html",
  ],
  theme: {
    extend: {
      colors: {
        primary: "#2d5016",
        secondary: "#8b4513",
        accent: "#d4a574",
        background: "#f5f0e8",
        surface: "#faf8f5",
        text: "#1a2f0a",
        "text-light": "#4a5d23",
        success: "#457528",
        warning: "#a84a2e",
        error: "#8b2500",
      },
      fontFamily: {
        heading: ["'Playfair Display'", "serif"],
        body: ["'Inter'", "sans-serif"],
        accent: ["'Caveat'", "cursive"],
      },
    },
  },
  plugins: [],
}
