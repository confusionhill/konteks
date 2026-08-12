/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}'],
  theme: {
    extend: {
      colors: {
        background: '#F8FAFC',
        surface: '#FFFFFF',
        primary: {
          DEFAULT: '#4F46E5', // Indigo 600
          hover: '#4338CA',   // Indigo 700
          light: '#EEF2FF',   // Indigo 50
        },
        secondary: {
          DEFAULT: '#7C3AED', // Violet 600
          light: '#F5F3FF',   // Violet 50
        },
        slate: {
          900: '#0F172A',
          800: '#1E293B',
          700: '#334155',
          600: '#475569',
          500: '#64748B',
          200: '#E2E8F0',
          100: '#F1F5F9',
          50: '#F8FAFC',
        }
      },
      fontFamily: {
        sans: ['Plus Jakarta Sans', 'system-ui', 'sans-serif'],
        mono: ['Geist Mono', 'Fira Code', 'monospace'],
      },
      boxShadow: {
        'soft-indigo': '0 4px 20px -2px rgba(79, 70, 229, 0.1)',
        'soft-indigo-hover': '0 12px 28px -4px rgba(79, 70, 229, 0.18), 0 8px 10px -6px rgba(79, 70, 229, 0.1)',
        'btn-indigo': '0 4px 14px 0 rgba(79, 70, 229, 0.35)',
        'glow-indigo': '0 0 25px rgba(79, 70, 229, 0.45)',
      },
      animation: {
        'float-slow': 'float 6s ease-in-out infinite',
        'pulse-subtle': 'pulse 4s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
      keyframes: {
        float: {
          '0%, 100%': { transform: 'translateY(0px)' },
          '50%': { transform: 'translateY(-8px)' },
        }
      }
    },
  },
  plugins: [],
}
