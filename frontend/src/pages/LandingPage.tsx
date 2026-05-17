import { Link } from 'react-router';

// Inline SVG decorations
const LeafLeft = () => (
  <svg className="absolute -left-4 top-0 w-8 h-20 text-primary/10 animate-gentle-pulse" viewBox="0 0 32 80" fill="currentColor">
    <path d="M16 0 C16 0 4 20 4 40 C4 51 11 60 16 60 C21 60 28 51 28 40 C28 20 16 0 16 0Z" />
    <path d="M16 60 L16 80" stroke="currentColor" strokeWidth="2" fill="none" />
  </svg>
);

const LeafRight = () => (
  <svg className="absolute -right-4 top-0 w-8 h-20 text-primary/10 animate-gentle-pulse" style={{ animationDelay: '1.5s' }} viewBox="0 0 32 80" fill="currentColor">
    <path d="M16 0 C16 0 4 20 4 40 C4 51 11 60 16 60 C21 60 28 51 28 40 C28 20 16 0 16 0Z" />
    <path d="M16 60 L16 80" stroke="currentColor" strokeWidth="2" fill="none" />
  </svg>
);

const Sparkle = ({ delay = 0, className = '' }: { delay?: number; className?: string }) => (
  <svg
    className={`absolute text-accent/60 animate-sparkle ${className}`}
    style={{ animationDelay: `${delay}s` }}
    width="12" height="12" viewBox="0 0 12 12" fill="currentColor"
  >
    <path d="M6 0 L7 4 L12 6 L7 8 L6 12 L5 8 L0 6 L5 4 Z" />
  </svg>
);

export default function LandingPage() {
  return (
    <div className="min-h-screen flex items-center justify-center relative overflow-hidden">
      {/* Skip link target for pages without a main element */}
      <span id="main-content" className="sr-only" />

      {/* Background decorative elements */}
      <div className="absolute inset-0 pointer-events-none overflow-hidden" aria-hidden="true">
        {/* Top-left vine */}
        <svg className="absolute -top-8 -left-8 w-40 h-40 text-primary/5" viewBox="0 0 160 160" fill="currentColor">
          <path d="M0 0 C40 20 60 60 40 100 C30 120 10 140 0 160" stroke="currentColor" strokeWidth="3" fill="none" />
          <path d="M20 30 C35 25 45 35 40 50 C35 65 20 60 15 50" fill="currentColor" opacity="0.5" />
          <path d="M30 70 C50 65 55 80 45 95 C35 110 20 100 15 85" fill="currentColor" opacity="0.5" />
        </svg>
        {/* Bottom-right vine */}
        <svg className="absolute -bottom-8 -right-8 w-40 h-40 text-primary/5" viewBox="0 0 160 160" fill="currentColor" style={{ transform: 'rotate(180deg)' }}>
          <path d="M0 0 C40 20 60 60 40 100 C30 120 10 140 0 160" stroke="currentColor" strokeWidth="3" fill="none" />
          <path d="M20 30 C35 25 45 35 40 50 C35 65 20 60 15 50" fill="currentColor" opacity="0.5" />
          <path d="M30 70 C50 65 55 80 45 95 C35 110 20 100 15 85" fill="currentColor" opacity="0.5" />
        </svg>
      </div>

      <main role="main" className="text-center relative animate-fade-in-up">
        {/* Heading with leaf decorations */}
        <div className="relative inline-block mb-2">
          <LeafLeft />
          <LeafRight />
          <h1 className="text-6xl md:text-7xl font-heading text-primary tracking-tight drop-shadow-sm">
            Our Library
          </h1>
        </div>

        {/* Tagline in Caveat font */}
        <p className="text-2xl md:text-3xl font-accent text-text-light mb-2 animate-fade-in" style={{ animationDelay: '0.2s' }}>
          ✦ A woodland fairy tale collection ✦
        </p>

        {/* Sparkle accents around tagline */}
        <Sparkle delay={0.5} className="top-0 left-1/4" />
        <Sparkle delay={1.2} className="top-0 right-1/4" />
        <Sparkle delay={0.8} className="bottom-0 left-1/3" />
        <Sparkle delay={1.5} className="bottom-0 right-1/3" />

        {/* Decorative divider */}
        <div className="vine-divider justify-center mb-8" aria-hidden="true">
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
            <path d="M8 1 C8 1 5 5 5 8 C5 9.7 6.3 11 8 11 C9.7 11 11 9.7 11 8 C11 5 8 1 8 1Z" />
          </svg>
        </div>

        {/* Login buttons */}
        <div className="flex flex-col sm:flex-row items-center justify-center gap-4 animate-fade-in" style={{ animationDelay: '0.4s' }}>
          <Link
            to="/login"
            className="group relative px-8 py-3.5 bg-primary text-white rounded-xl
                       shadow-lg shadow-primary/20 hover:shadow-xl hover:shadow-primary/30
                       hover:-translate-y-0.5 active:translate-y-0
                       font-medium text-lg tracking-wide
                       transition-all duration-200 ease-in-out"
            aria-label="Sign in as administrator"
          >
            <span className="flex items-center gap-2">
              {/* Key icon */}
              <svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" />
              </svg>
              Admin Login
            </span>
          </Link>
          <Link
            to="/guest-login"
            className="group relative px-8 py-3.5 bg-secondary text-white rounded-xl
                       shadow-lg shadow-secondary/20 hover:shadow-xl hover:shadow-secondary/30
                       hover:-translate-y-0.5 active:translate-y-0
                       font-medium text-lg tracking-wide
                       transition-all duration-200 ease-in-out"
            aria-label="Sign in as guest to browse the collection"
          >
            <span className="flex items-center gap-2">
              {/* Leaf icon */}
              <svg className="w-5 h-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M17 8C8 10 5.9 16.17 3.82 21.34l1.89.66.95-2.3c.48.17.98.3 1.34.3C19 20 22 3 22 3c-1 2-8 2.25-13 3.25S2 11.5 2 13.5s1.75 3.75 1.75 3.75C7 8 17 8 17 8z" />
              </svg>
              Guest Login
            </span>
          </Link>
        </div>

        {/* Bottom decorative text */}
        <p className="mt-12 text-sm text-text-light/50 font-accent text-lg">
          🍄 Step into the enchanted forest of stories 🍄
        </p>
      </main>
    </div>
  );
}
