/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      fontFamily: {
        sans: ['"Plus Jakarta Sans"', "system-ui", "sans-serif"],
        mono: ['"JetBrains Mono"', "monospace"],
      },
      colors: {
        background: "#F8FAFC",
        surface: "#FFFFFF",
        primary: {
          DEFAULT: "#4F46E5",
          50: "#EEF2FF",
          100: "#E0E7FF",
          600: "#4F46E5",
          700: "#4338CA",
          900: "#312E81",
        },
        secondary: {
          DEFAULT: "#7C3AED",
          600: "#7C3AED",
        },
        success: "#10B981",
      },
      boxShadow: {
        soft: "0 4px 20px -2px rgba(79, 70, 229, 0.1)",
        lifted: "0 10px 25px -5px rgba(79, 70, 229, 0.15), 0 8px 10px -6px rgba(79, 70, 229, 0.1)",
        button: "0 4px 14px 0 rgba(79, 70, 229, 0.3)",
      },
      borderRadius: {
        xl: "12px",
        lg: "8px",
      },
    },
  },
  plugins: [],
};
