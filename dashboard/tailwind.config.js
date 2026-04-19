/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{html,js,svelte,ts}'],
  theme: {
    extend: {
      colors: {
        // Nexos palette — dark-mode-first dashboard
        bg: '#0b0f14',
        panel: '#121820',
        border: '#1e2630',
        accent: '#4ade80',
        warn: '#f59e0b',
        danger: '#ef4444'
      }
    }
  },
  plugins: []
};
