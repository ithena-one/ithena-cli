/* eslint-disable @typescript-eslint/no-explicit-any */
import { useState, useEffect } from 'react';
import ithenaLogo from '@/assets/ithena-logo.svg';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";

const ITHENA_PLATFORM_URL_FALLBACK = "https://ithena.one";

interface AuthStatus {
  authenticated: boolean;
  platformURL: string;
}

interface CliVersionInfo {
  version: string;
}

export default function Header() {
  const [authStatus, setAuthStatus] = useState<AuthStatus | null>(null);
  const [isLoadingAuth, setIsLoadingAuth] = useState(true);
  const [currentCliVersion, setCurrentCliVersion] = useState<string | null>(null);
  const [latestCliVersion, setLatestCliVersion] = useState<string | null>(null);
  const [updateAvailable, setUpdateAvailable] = useState(false);
  const [isLoadingVersion, setIsLoadingVersion] = useState(true);
  const [isUpdateDialogOpen, setIsUpdateDialogOpen] = useState(false);
  const [copiedCommand, setCopiedCommand] = useState(false);

  const compareVersions = (v1: string, v2: string): number => {
    const normalize = (v: string) => v.replace(/^v/, '').split('.').map(Number);
    const parts1 = normalize(v1);
    const parts2 = normalize(v2);

    for (let i = 0; i < Math.max(parts1.length, parts2.length); i++) {
      const num1 = parts1[i] || 0;
      const num2 = parts2[i] || 0;
      if (num1 < num2) return -1; // v1 is older
      if (num1 > num2) return 1;  // v1 is newer
    }
    return 0; // versions are equal
  };

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

  useEffect(() => {
    const fetchCliVersion = async () => {
      setIsLoadingVersion(true);
      try {
        const response = await fetch('/api/version');
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data: CliVersionInfo = await response.json();
        if (data.version && data.version.startsWith('v')) {
          setCurrentCliVersion(data.version);
        } else {
          setCurrentCliVersion(null); // Treat empty or non-standard versions as 'unversioned' for this UI
        }
      } catch (error: any) {
        console.error('Failed to fetch CLI version:', error);
        setCurrentCliVersion(null);
      } finally {
        setIsLoadingVersion(false);
      }
    };
    fetchCliVersion();
  }, []);

  useEffect(() => {
    // Only proceed if currentCliVersion is a valid-looking version string (e.g., starts with 'v')
    if (!currentCliVersion || !currentCliVersion.startsWith('v')) {
      setLatestCliVersion(null); // Ensure no old latest version is shown
      setUpdateAvailable(false); // Ensure no update notification for dev/unversioned builds
      return;
    }

    const fetchLatestRelease = async () => {
      // setIsLoadingGitHubRelease(true); // Removed
      try {
        const response = await fetch('https://api.github.com/repos/ithena-one/ithena-cli/releases/latest');
        if (!response.ok) {
          throw new Error(`GitHub API error! status: ${response.status}`);
        }
        const data = await response.json();
        if (data && data.tag_name) {
          setLatestCliVersion(data.tag_name);
          if (currentCliVersion && compareVersions(currentCliVersion, data.tag_name) < 0) { // Ensure currentCliVersion is not null
            setUpdateAvailable(true);
          }
        } else {
          setLatestCliVersion(null);
        }
      } catch (error: any) {
        console.error('Failed to fetch latest GitHub release:', error);
        setLatestCliVersion(null);
      } finally {
        // setIsLoadingGitHubRelease(false); // Removed
      }
    };

    fetchLatestRelease();
  }, [currentCliVersion]);

  useEffect(() => {
    (window as any).showTestUpdateNotification = (show: boolean, testVersion: string = "v99.9.9") => {
      if (show) {
        setLatestCliVersion(testVersion);
        setUpdateAvailable(true);
      } else {
        setLatestCliVersion(null);
        setUpdateAvailable(false);
      }
    };
    return () => {
      delete (window as any).showTestUpdateNotification;
    };
  }, []);

  const platformUrl = authStatus?.platformURL || ITHENA_PLATFORM_URL_FALLBACK;

  return (
    // Uses Tailwind utilities and respects potential dark mode via CSS variables from index.css
    <header className="sticky top-0 z-50 w-full border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="flex h-16 items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-2">
              <img src={ithenaLogo} alt="Ithena Logo" className="h-8 w-8" />
              <div className="flex flex-col items-start">
                <span className="font-semibold text-foreground">Ithena Local Logs</span>
                {isLoadingVersion ? (
                  <span className="text-xs text-muted-foreground">Loading version...</span>
                ) : currentCliVersion ? ( // currentCliVersion is now either like "v1.2.3" or null
                  <Badge variant="outline" className="text-xs">CLI: {currentCliVersion}</Badge>
                ) : (
                  <Badge variant="secondary" className="text-xs">CLI: Dev Build</Badge> // More specific for null/empty version
                )}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-4">
            {updateAvailable && latestCliVersion && (
              <Dialog open={isUpdateDialogOpen} onOpenChange={setIsUpdateDialogOpen}>
                <DialogTrigger asChild>
                  <Badge variant="default" className="bg-blue-500 hover:bg-blue-500/90 text-white select-none cursor-pointer">
                    Update available: {latestCliVersion}
                  </Badge>
                </DialogTrigger>
                <DialogContent className="sm:max-w-md">
                  <DialogHeader>
                    <DialogTitle>Update Available: {latestCliVersion}</DialogTitle>
                    <DialogDescription>
                      A new version of the Ithena CLI is available.
                    </DialogDescription>
                  </DialogHeader>
                  <div className="space-y-4 py-2">
                    <div>
                      <h4 className="font-medium text-sm mb-1">Linux / macOS:</h4>
                      <p className="text-sm text-muted-foreground">
                        Run the following command in your terminal:
                      </p>
                      <pre className="mt-2 p-2 bg-muted rounded-md text-xs overflow-x-auto max-w-full">
                        <code className="whitespace-pre-wrap break-all">curl -sfL https://raw.githubusercontent.com/ithena-one/ithena-cli/main/install.sh | bash</code>
                      </pre>
                      <Button 
                        variant="outline" 
                        size="sm" 
                        className="mt-2"
                        onClick={() => {
                          navigator.clipboard.writeText("curl -sfL https://raw.githubusercontent.com/ithena-one/ithena-cli/main/install.sh | bash");
                          setCopiedCommand(true);
                          setTimeout(() => setCopiedCommand(false), 2000);
                        }}
                      >
                        {copiedCommand ? "Copied!" : "Copy Command"}
                      </Button>
                    </div>
                    <div>
                      <h4 className="font-medium text-sm mb-1">Windows:</h4>
                      <p className="text-sm text-muted-foreground">
                        Please download the latest release from the GitHub releases page.
                      </p>
                    </div>
                  </div>
                  <DialogFooter className="sm:justify-between">
                    <Button variant="ghost" asChild>
                      <a href="https://github.com/ithena-one/ithena-cli/releases" target="_blank" rel="noopener noreferrer">
                        Go to Releases Page
                      </a>
                    </Button>
                    <DialogClose asChild>
                      <Button type="button" variant="secondary">
                        Close
                      </Button>
                    </DialogClose>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            )}
            {isLoadingAuth ? (
              <span className="text-sm text-muted-foreground">Loading auth...</span>
            ) : authStatus ? (
              authStatus.authenticated ? (
                <Badge variant="default" className="bg-green-500 hover:bg-green-500/90 text-white select-none">
                  Authenticated
                </Badge>
              ) : (
                <Badge variant="destructive" className="select-none">
                  Unauthenticated
                </Badge>
              )
            ) : (
              <Badge variant="outline" className="select-none">
                Status Unknown
              </Badge>
            )}
            <Button variant="secondary" size="sm" asChild>
              <a
                href={platformUrl}
                target="_blank"
                rel="noopener noreferrer"
              >
                Go to Platform
              </a>
            </Button>
          </div>
        </div>
      </div>
    </header>
  );
} 