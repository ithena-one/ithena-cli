/* eslint-disable @typescript-eslint/no-explicit-any */
import { useState, useEffect, useCallback } from 'react';
import './index.css'; // We might need to adjust or remove this later

const ITHENA_PLATFORM_URL = "https://app.ithena.io";

interface AuthStatus {
  authenticated: boolean;
  platformURL: string;
}

interface LogEntry {
  id: string;
  timestamp: string;
  tool_name?: string;
  mcp_method?: string;
  status: string;
  duration_ms?: number;
  [key: string]: any; // Allow other properties for detailed view flexibility
}

interface LogsApiResponse {
  logs: LogEntry[];
  total_count: number;
  page: number;
  limit: number;
}

function escapeHtml(unsafe: any): string {
  if (unsafe === null || typeof unsafe === 'undefined') return '-';
  if (typeof unsafe !== 'string') return String(unsafe);
  return unsafe
       .replace(/&/g, "&amp;")
       .replace(/</g, "&lt;")
       .replace(/>/g, "&gt;")
       .replace(/"/g, "&quot;")
       .replace(/'/g, "&#039;");
}

function App() {
  const [authStatus, setAuthStatus] = useState<AuthStatus | null>(null);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [isLoadingLogs, setIsLoadingLogs] = useState(true);
  const [errorLoadingLogs, setErrorLoadingLogs] = useState<string | null>(null);
  const [selectedLog, setSelectedLog] = useState<LogEntry | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  // Filter states
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [toolNameFilter, setToolNameFilter] = useState<string>("");
  const [searchTermFilter, setSearchTermFilter] = useState<string>("");

  // TODO: Add state for pagination (currentPage, totalPages, logsPerPage)

  const fetchLogs = useCallback(async (page = 1, limit = 20) => {
    setIsLoadingLogs(true);
    setErrorLoadingLogs(null);
    
    let apiUrl = `/api/logs?page=${page}&limit=${limit}`;
    if (statusFilter) apiUrl += `&status=${encodeURIComponent(statusFilter)}`;
    if (toolNameFilter) apiUrl += `&tool_name=${encodeURIComponent(toolNameFilter)}`;
    if (searchTermFilter) apiUrl += `&search=${encodeURIComponent(searchTermFilter)}`;

    try {
      const response = await fetch(apiUrl);
      if (!response.ok) {
        throw new Error(`API Error: ${response.status} ${response.statusText}`);
      }
      const data: LogsApiResponse = await response.json();
      setLogs(data.logs || []);
      // TODO: Update pagination state (data.total_count, data.page, data.limit)
    } catch (error: any) {
      console.error("Failed to fetch logs:", error);
      setErrorLoadingLogs(error.message || "Failed to retrieve logs");
      setLogs([]);
    } finally {
      setIsLoadingLogs(false);
    }
  }, [statusFilter, toolNameFilter, searchTermFilter]); // Add filter states to dependencies

  useEffect(() => {
    const fetchAuth = async () => {
      try {
        const response = await fetch('/api/auth/status');
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
        const data: AuthStatus = await response.json();
        setAuthStatus(data);
      } catch (error) {
        console.error('Failed to fetch auth status:', error);
        setAuthStatus({ authenticated: false, platformURL: ITHENA_PLATFORM_URL });
      }
    };

    fetchAuth();
    fetchLogs(); // Initial log fetch
  }, [fetchLogs]);

  // Initial fetch and re-fetch when filter-dependent fetchLogs changes
  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  const handleViewDetails = async (logId: string) => {
    try {
      const response = await fetch(`/api/logs/${logId}`);
      if (!response.ok) {
        throw new Error(`Failed to fetch log details: ${response.status} ${response.statusText}`);
      }
      const logDetailData: LogEntry = await response.json();
      setSelectedLog(logDetailData);
      setIsModalOpen(true);
    } catch (error: any) {
      console.error("Error fetching log details:", error);
      alert(`Could not load log details: ${error.message}`);
    }
  };

  const closeModal = () => {
    setIsModalOpen(false);
    setSelectedLog(null);
  };

  const handleApplyFilters = () => {
    fetchLogs(1); // Reset to page 1 when applying filters
  };

  const handleFilterKeyPress = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter') {
      handleApplyFilters();
    }
  };

  return (
    <div className="bg-gray-100 text-gray-800 font-sans min-h-screen">
      <header id="page-header" className="bg-ithena-gray text-white p-4 shadow-md">
        <div className="container mx-auto flex justify-between items-center">
          <div id="auth-status-container" className="flex items-center">
            <span id="auth-status" className="mr-4">
              {authStatus === null ? 'Loading auth status...' :
               authStatus.authenticated ? 'Status: Authenticated' : 'Status: Unauthenticated'}
            </span>
            <button
              id="auth-action-button"
              className="bg-ithena-blue hover:bg-blue-700 text-white font-bold py-2 px-4 rounded transition duration-150 ease-in-out"
              onClick={() => {
                if (authStatus?.authenticated) {
                  window.open(authStatus.platformURL, '_blank');
                } else {
                  alert('To log in, please run `ithena-cli auth login` in your terminal.');
                }
              }}
              disabled={authStatus === null}
            >
              {authStatus === null ? 'Loading...' :
               authStatus.authenticated ? 'View My Account on Ithena' : 'Login with Ithena'}
            </button>
          </div>
          <a
            href={ITHENA_PLATFORM_URL}
            target="_blank"
            id="platform-link"
            className="text-ithena-blue hover:text-blue-400 hover:underline"
            rel="noopener noreferrer"
          >
            Go to Ithena Platform
          </a>
        </div>
      </header>

      <div className="container mx-auto p-4 mt-6">
        <h1 className="text-3xl font-bold text-ithena-gray mb-6 text-center">Ithena Local MCP Logs</h1>

        <div id="filters" className="flex flex-wrap items-center gap-4 mb-6 p-4 bg-white rounded-lg shadow">
          <div>
            <label htmlFor="status-filter" className="block text-sm font-medium text-gray-700 mr-2">Status:</label>
            <select 
              id="status-filter"
              className="p-2 border border-gray-300 rounded-md shadow-sm focus:ring-ithena-blue focus:border-ithena-blue sm:text-sm"
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
            >
              <option value="">All Statuses</option>
              <option value="success">Success</option>
              <option value="failure">Failure</option>
            </select>
          </div>

          <div>
            <label htmlFor="tool-name-filter" className="block text-sm font-medium text-gray-700 mr-2">Tool Name:</label>
            <input 
              type="text"
              id="tool-name-filter"
              placeholder="Filter by Tool Name..."
              className="p-2 border border-gray-300 rounded-md shadow-sm focus:ring-ithena-blue focus:border-ithena-blue sm:text-sm w-full md:w-auto"
              value={toolNameFilter}
              onChange={(e) => setToolNameFilter(e.target.value)}
              onKeyPress={handleFilterKeyPress}
            />
          </div>

          <div>
            <label htmlFor="search-term-filter" className="block text-sm font-medium text-gray-700 mr-2">Search:</label>
            <input 
              type="text"
              id="search-term-filter"
              placeholder="Global Search..."
              className="p-2 border border-gray-300 rounded-md shadow-sm focus:ring-ithena-blue focus:border-ithena-blue sm:text-sm w-full md:w-auto"
              value={searchTermFilter}
              onChange={(e) => setSearchTermFilter(e.target.value)}
              onKeyPress={handleFilterKeyPress}
            />
          </div>

          <button 
            onClick={handleApplyFilters}
            className="self-end py-2 px-4 bg-ithena-blue hover:bg-blue-700 text-white font-semibold rounded-md shadow-sm transition duration-150 ease-in-out"
          >
            Apply Filters
          </button>
        </div>

        <div id="log-table-container" className="bg-white rounded-lg shadow overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Timestamp</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Tool/Method</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Duration (ms)</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Details</th>
              </tr>
            </thead>
            <tbody id="log-table-body" className="bg-white divide-y divide-gray-200">
              {isLoadingLogs && (
                <tr><td colSpan={5} className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-center">Loading logs...</td></tr>
              )}
              {errorLoadingLogs && (
                 <tr><td colSpan={5} className="px-6 py-4 whitespace-nowrap text-sm text-red-500 text-center">Error loading logs: {errorLoadingLogs}</td></tr>
              )}
              {!isLoadingLogs && !errorLoadingLogs && logs.length === 0 && (
                <tr><td colSpan={5} className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-center">No logs found.</td></tr>
              )}
              {!isLoadingLogs && !errorLoadingLogs && logs.map(log => {
                const timestamp = new Date(log.timestamp).toLocaleString();
                const toolOrMethod = log.tool_name || log.mcp_method || '-';
                const duration = log.duration_ms !== undefined && log.duration_ms !== null ? `${log.duration_ms}ms` : '-';
                let statusClass = 'text-gray-700';
                if (log.status?.toLowerCase() === 'success') {
                    statusClass = 'text-green-600 font-semibold';
                } else if (log.status?.toLowerCase() === 'failure') {
                    statusClass = 'text-red-600 font-semibold';
                }

                return (
                  <tr key={log.id}>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{timestamp}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{escapeHtml(toolOrMethod)}</td>
                    <td className={`px-6 py-4 whitespace-nowrap text-sm ${statusClass}`}>{escapeHtml(log.status)}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600 text-right">{duration}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600 text-center">
                        <button
                          className="view-details-btn bg-ithena-blue hover:bg-blue-600 text-white py-1 px-3 rounded-md text-xs font-medium transition duration-150 ease-in-out"
                          onClick={() => handleViewDetails(log.id)}
                        >
                            View
                        </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>

        <div id="pagination" className="mt-6 p-4 bg-white rounded-lg shadow flex justify-center items-center space-x-2">
          <p className="text-gray-600">Pagination will appear here. (TODO)</p>
        </div>

        {isModalOpen && selectedLog && (
          <div
            id="log-detail-modal"
            className="fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full flex items-center justify-center z-50"
            onClick={closeModal} // Close modal if backdrop is clicked
          >
            <div
              className="modal-content bg-white p-6 rounded-lg shadow-xl w-11/12 md:w-3/4 lg:w-1/2 max-h-[80vh] overflow-y-auto"
              onClick={(e) => e.stopPropagation()} // Prevent click inside modal from closing it
            >
              <div className="flex justify-between items-center mb-4">
                <h2 className="text-2xl font-semibold text-ithena-gray">Log Details</h2>
                <span
                  className="close-button text-gray-700 hover:text-red-500 text-3xl font-bold cursor-pointer"
                  onClick={closeModal}
                >
                  &times;
                </span>
              </div>
              <pre id="log-detail-content" className="bg-gray-100 p-4 rounded text-sm text-gray-700 whitespace-pre-wrap break-all">
                {selectedLog ? JSON.stringify(selectedLog, null, 2) : 'No details available.'}
              </pre>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default App;