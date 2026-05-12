export default function LandingPage() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="text-center">
        <h1 className="text-5xl font-heading text-primary mb-4">Our Library</h1>
        <p className="text-xl text-text-light mb-8">A woodland fairy tale collection</p>
        <div className="space-x-4">
          <a href="/login" className="px-6 py-3 bg-primary text-white rounded-lg hover:bg-opacity-90 transition">Admin Login</a>
          <a href="/guest-login" className="px-6 py-3 bg-secondary text-white rounded-lg hover:bg-opacity-90 transition">Guest Login</a>
        </div>
      </div>
    </div>
  );
}
