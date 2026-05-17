/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      colors: {
        bg:       'var(--bg)',
        surface:  'var(--surface)',
        elevated: 'var(--surface-high)',
        border:   'var(--border)',
        accent:   'var(--accent)',
        danger:   'var(--danger)',
        warning:  'var(--warning)',
        online:   'var(--online)',
        hi:       'var(--text-1)',
        lo:       'var(--text-2)',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
      borderRadius: {
        card: '12px',
        btn:  '8px',
      },
      boxShadow: {
        accent: '0 0 20px var(--accent-glow)',
        card:   '0 4px 24px rgba(0,0,0,0.4)',
      },
      keyframes: {
        shimmer: {
          '0%':   { backgroundPosition: '-400px 0' },
          '100%': { backgroundPosition:  '400px 0' },
        },
        'fade-in': {
          from: { opacity: 0, transform: 'translateY(6px)' },
          to:   { opacity: 1, transform: 'translateY(0)' },
        },
        'slide-up': {
          from: { opacity: 0, transform: 'translateY(100%)' },
          to:   { opacity: 1, transform: 'translateY(0)' },
        },
        'slide-in-right': {
          from: { transform: 'translateX(100%)' },
          to:   { transform: 'translateX(0)' },
        },
      },
      animation: {
        shimmer:          'shimmer 1.4s ease infinite',
        'fade-in':        'fade-in 0.2s ease',
        'slide-up':       'slide-up 0.3s ease',
        'slide-in-right': 'slide-in-right 0.25s ease',
      },
    },
  },
  plugins: [],
}
