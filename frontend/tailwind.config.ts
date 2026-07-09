import type { Config } from "tailwindcss";

export default {
	darkMode: ["class"],
	content: [
		"./pages/**/*.{ts,tsx}",
		"./components/**/*.{ts,tsx}",
		"./app/**/*.{ts,tsx}",
		"./src/**/*.{ts,tsx}",
	],
	prefix: "",
	theme: {
		container: {
			center: true,
			padding: '2rem',
			screens: {
				'2xl': '1400px'
			}
		},
		screens: {
			'xs': '475px',
			'sm': '640px',
			'md': '768px',
			'lg': '1024px',
			'xl': '1280px',
			'2xl': '1536px'
		},
		extend: {
			fontFamily: {
				// INDUSTRIAL LUXURY Typography - Breaking Anti-Patterns
				display: ['Manrope', 'ui-sans-serif', 'system-ui', 'sans-serif'],
				sans: ['Inter', 'ui-sans-serif', 'system-ui', 'sans-serif'],
				financial: ['JetBrains Mono', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'Consolas', 'monospace'],
			},
			colors: {
				employee: {
					DEFAULT: '#00B14F',
					50: '#e6f9ef',
					100: '#b3efd1',
					200: '#80e5b3',
					300: '#4ddb95',
					400: '#26d67d',
					500: '#00B14F',
					600: '#009e45',
					700: '#007a37',
					800: '#005729',
					900: '#00331a',
				},
				border: 'hsl(var(--border))',
				input: 'hsl(var(--input))',
				ring: 'hsl(var(--ring))',
				background: 'hsl(var(--background))',
				foreground: 'hsl(var(--foreground))',
				primary: {
					DEFAULT: 'hsl(var(--primary))',
					foreground: 'hsl(var(--primary-foreground))'
				},
				secondary: {
					DEFAULT: 'hsl(var(--secondary))',
					foreground: 'hsl(var(--secondary-foreground))'
				},
				destructive: {
					DEFAULT: 'hsl(var(--destructive))',
					foreground: 'hsl(var(--destructive-foreground))'
				},
				muted: {
					DEFAULT: 'hsl(var(--muted))',
					foreground: 'hsl(var(--muted-foreground))'
				},
				accent: {
					DEFAULT: 'hsl(var(--accent))',
					foreground: 'hsl(var(--accent-foreground))'
				},
				popover: {
					DEFAULT: 'hsl(var(--popover))',
					foreground: 'hsl(var(--popover-foreground))'
				},
				card: {
					DEFAULT: 'hsl(var(--card))',
					foreground: 'hsl(var(--card-foreground))'
				},
				sidebar: {
					DEFAULT: 'hsl(var(--sidebar-background))',
					foreground: 'hsl(var(--sidebar-foreground))',
					primary: 'hsl(var(--sidebar-primary))',
					'primary-foreground': 'hsl(var(--sidebar-primary-foreground))',
					accent: 'hsl(var(--sidebar-accent))',
					'accent-foreground': 'hsl(var(--sidebar-accent-foreground))',
					border: 'hsl(var(--sidebar-border))',
					ring: 'hsl(var(--sidebar-ring))'
				},
				success: 'hsl(var(--success))',
				warning: 'hsl(var(--warning))',
				info: 'hsl(var(--info))',
			},
			backgroundImage: {
				'gradient-navy': 'linear-gradient(135deg, hsl(220 90% 25%) 0%, hsl(220 80% 35%) 100%)',
				'gradient-teal': 'linear-gradient(135deg, hsl(180 70% 45%) 0%, hsl(180 65% 40%) 100%)',
				'gradient-subtle': 'linear-gradient(180deg, hsl(220 20% 98%) 0%, hsl(220 15% 96%) 100%)',
			},
			boxShadow: {
				'navy': '0 8px 32px hsl(220 90% 25% / 0.25)',
				'card-elevated': '0 1px 3px hsl(220 20% 90%), 0 10px 40px -10px hsl(220 30% 85%), 0 0 0 1px hsl(220 10% 90%) inset',
			},
			keyframes: {
				// Staggered page load animations
				'hero-reveal': {
					'0%': { opacity: '0', transform: 'translateY(20px) scale(0.98)' },
					'100%': { opacity: '1', transform: 'translateY(0) scale(1)' },
				},
				'fade-in-up': {
					'0%': { opacity: '0', transform: 'translateY(10px)' },
					'100%': { opacity: '1', transform: 'translateY(0)' },
				},
				'slide-in-right': {
					'0%': { opacity: '0', transform: 'translateX(-20px)' },
					'100%': { opacity: '1', transform: 'translateX(0)' },
				},
				// Micro-interactions
				'button-press': {
					'0%': { transform: 'scale(1)' },
					'50%': { transform: 'scale(0.98) translateY(1px)' },
					'100%': { transform: 'scale(1) translateY(0)' },
				},
				// Grain texture animation
				'grain-shift': {
					'0%, 100%': { transform: 'translate(0, 0)' },
					'10%': { transform: 'translate(-5%, -10%)' },
					'20%': { transform: 'translate(-15%, 5%)' },
					'30%': { transform: 'translate(7%, -25%)' },
					'40%': { transform: 'translate(-5%, 25%)' },
					'50%': { transform: 'translate(-15%, 10%)' },
					'60%': { transform: 'translate(15%, 0%)' },
					'70%': { transform: 'translate(0%, 15%)' },
					'80%': { transform: 'translate(3%, 35%)' },
					'90%': { transform: 'translate(-10%, 10%)' },
				},
				// Status dot pulse (pending approval indicator)
				'status-pulse': {
					'0%, 100%': { opacity: '1', transform: 'scale(1)' },
					'50%': { opacity: '0.5', transform: 'scale(1.4)' },
				},
				// Skeleton shimmer
				'shimmer': {
					'0%': { backgroundPosition: '-200% 0' },
					'100%': { backgroundPosition: '200% 0' },
				},
				// Toast slide-in with elastic bounce
				'toast-slide-in': {
					'0%': { opacity: '0', transform: 'translateX(110%) scale(0.95)' },
					'60%': { opacity: '1', transform: 'translateX(-6%) scale(1.01)' },
					'80%': { transform: 'translateX(3%) scale(0.99)' },
					'100%': { opacity: '1', transform: 'translateX(0) scale(1)' },
				},
				// Sidebar active pill glide (handled via CSS transition, this is a fallback)
				'pill-appear': {
					'0%': { opacity: '0', transform: 'scaleY(0.5)' },
					'100%': { opacity: '1', transform: 'scaleY(1)' },
				},
				// Data point pulse on chart hover
				'dot-pulse': {
					'0%': { transform: 'scale(1)', opacity: '1' },
					'50%': { transform: 'scale(1.6)', opacity: '0.7' },
					'100%': { transform: 'scale(1)', opacity: '1' },
				},
			},
			animation: {
				'hero-reveal': 'hero-reveal 0.6s cubic-bezier(0.2, 0.8, 0.2, 1) forwards',
				'fade-in-up': 'fade-in-up 0.4s ease-out forwards',
				'slide-in-right': 'slide-in-right 0.3s ease-out forwards',
				'button-press': 'button-press 0.2s ease-out',
				'grain-shift': 'grain-shift 8s steps(10) infinite',
				'status-pulse': 'status-pulse 2s ease-in-out infinite',
				'shimmer': 'shimmer 1.6s linear infinite',
				'toast-slide-in': 'toast-slide-in 0.45s cubic-bezier(0.2, 0.8, 0.2, 1) forwards',
				'pill-appear': 'pill-appear 0.2s ease-out forwards',
				'dot-pulse': 'dot-pulse 0.6s ease-in-out',
			},
			transitionDelay: {
				'50': '50ms',
				'100': '100ms',
				'150': '150ms',
				'200': '200ms',
				'250': '250ms',
				'300': '300ms',
				'350': '350ms',
				'400': '400ms',
				'450': '450ms',
				'500': '500ms',
			},
			fontSize: {
				'xs': ['0.6875rem', { lineHeight: '1.4' }],    // 11px
				'sm': ['0.75rem', { lineHeight: '1.4' }],     // 12px
				'base': ['0.8125rem', { lineHeight: '1.5' }],  // 13px
				'lg': ['0.875rem', { lineHeight: '1.4' }],    // 14px
				'xl': ['1rem', { lineHeight: '1.3' }],        // 16px
				'2xl': ['1.125rem', { lineHeight: '1.3' }],   // 18px
				'3xl': ['1.25rem', { lineHeight: '1.25' }],   // 20px
				'4xl': ['1.5rem', { lineHeight: '1.2' }],     // 24px
				'5xl': ['1.75rem', { lineHeight: '1.15' }],   // 28px
			},
			fontWeight: {
				thin: '100',
				extralight: '200',
				light: '300',
				normal: '400',
				medium: '500',
				semibold: '600',
				bold: '700',
				extrabold: '800',
				black: '900',
			},
			lineHeight: {
				none: '1',
				tight: '1.15',
				snug: '1.3',
				normal: '1.4',
				relaxed: '1.5',
				loose: '1.65',
			},
			letterSpacing: {
				tighter: '0em',
				tight: '0em',
				normal: '0em',
				wide: '0.025em',
				wider: '0.05em',
				widest: '0.1em',
			},
		}
	},
	plugins: [
		require("tailwindcss-animate"),
		require("@tailwindcss/typography"),
	],
} satisfies Config;
