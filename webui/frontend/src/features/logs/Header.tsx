/* eslint-disable @typescript-eslint/no-explicit-any */
import { useState, useEffect } from 'react';
import ithenaLogo from '@/assets/ithena-logo.svg';
const ITHENA_PLATFORM_URL_FALLBACK = "https://ithena.one";

interface AuthStatus {
  authenticated: boolean;
  platformURL: string;
}

export default function Header() {
  const [authStatus, setAuthStatus] = useState<AuthStatus | null>(null);
  const [isLoadingAuth, setIsLoadingAuth] = useState(true);

  useEffect(() => {
    const fetchAuth = async () => {
      setIsLoadingAuth(true);
      try {
        const response = await fetch('/api/auth/status');
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data: AuthStatus = await response.json();
        setAuthStatus(data);
      } catch (error: any) {
        console.error('Failed to fetch auth status:', error);
        setAuthStatus({ authenticated: false, platformURL: ITHENA_PLATFORM_URL_FALLBACK });
      } finally {
        setIsLoadingAuth(false);
      }
    };
    fetchAuth();
  }, []);

  const platformUrl = authStatus?.platformURL || ITHENA_PLATFORM_URL_FALLBACK;

  const baseButtonClasses = "inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50";
  const platformButtonClasses = `${baseButtonClasses} bg-primary text-primary-foreground hover:bg-primary/90 px-4 py-2 h-9`; // Assumes bg-primary is dark, text-primary-foreground is light

  return (
    // Uses Tailwind utilities and respects potential dark mode via CSS variables from index.css
    <header className="sticky top-0 z-50 w-full border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="flex h-16 items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-2">
              <img src={ithenaLogo} alt="Ithena Logo" className="h-8 w-8" /> 
              <span className="font-semibold text-foreground">Ithena Local Logs</span>
            </div>
          </div>
          <div className="flex items-center gap-4">
            {isLoadingAuth ? (
              <span className="text-sm text-muted-foreground">Loading auth...</span>
            ) : authStatus ? (
              <span 
                className={`text-xs px-2.5 py-1 rounded-full font-semibold ${ 
                  authStatus.authenticated 
                    ? 'bg-green-500 text-black'
                    : 'bg-destructive text-black' 
                }`}
              >
                {authStatus.authenticated ? 'Authenticated' : 'Unauthenticated'}
              </span>
            ) : (
              <span className="text-xs px-2.5 py-1 rounded-full font-semibold bg-yellow-400 text-yellow-800">
                Status Unknown
              </span>
            )}
            <a
              href={platformUrl}
              target="_blank"
              rel="noopener noreferrer"
              className={platformButtonClasses} 
            >
              Go to Platform
            </a>
          </div>
        </div>
      </div>
    </header>
  );
} 