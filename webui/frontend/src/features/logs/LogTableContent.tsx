import { type LogEntry, type ColumnVisibilityState } from './types';

// Helper to safely escape HTML content (can be moved to a utils file)
function escapeHtml(unsafe: unknown): string {
  if (unsafe === null || typeof unsafe === 'undefined') return '-';
  if (typeof unsafe !== 'string') return String(unsafe);
  return unsafe
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

interface LogTableContentProps {
  logs: LogEntry[];
  isLoading: boolean;
  columnVisibility: ColumnVisibilityState;
  onOpenModal: (logId: string) => void;
  error?: Error | null; // Optional error display
}

export default function LogTableContent({
  logs,
  isLoading,
  columnVisibility,
  onOpenModal,
  error,
}: LogTableContentProps) {

  const getFormattedTimestamp = (isoDate: string) => {
    try {
      return new Date(isoDate).toLocaleString();
    } catch {
      return 'Invalid Date';
    }
  };

  const renderTableHeaders = () => {
    const headers = [];
    if (columnVisibility.timestamp) headers.push(<th key="timestamp" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Timestamp</th>);
    
    if (columnVisibility.tool_name) {
       headers.push(<th key="tool_name" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Tool</th>);
    }
    if (columnVisibility.mcp_method) {
      headers.push(<th key="mcp_method" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Method</th>);
    }

    if (columnVisibility.target_server_alias) headers.push(<th key="target_server_alias" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">MCP Host</th>);
    if (columnVisibility.status) headers.push(<th key="status" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>);
    if (columnVisibility.duration_ms) headers.push(<th key="duration" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Duration (ms)</th>);
    if (columnVisibility.id) headers.push(<th key="log_id" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Log ID</th>);
    headers.push(<th key="details" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Details</th>);
    return <tr>{headers}</tr>;
  };

  const renderTableBody = () => {
    if (isLoading) {
      return (
        <tr>
          <td colSpan={Object.values(columnVisibility).filter(v => v).length + 1} className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-center">
            Loading logs...
          </td>
        </tr>
      );
    }

    if (error) {
      return (
        <tr>
          <td colSpan={Object.values(columnVisibility).filter(v => v).length + 1} className="px-6 py-4 whitespace-nowrap text-sm text-red-500 text-center">
            Error loading logs: {error.message}
          </td>
        </tr>
      );
    }

    if (logs.length === 0) {
      return (
        <tr>
          <td colSpan={Object.values(columnVisibility).filter(v => v).length + 1} className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-center">
            No logs found.
          </td>
        </tr>
      );
    }

    return logs.map((log) => {
      let statusClass = 'text-gray-700';
      if (log.status?.toLowerCase() === 'success') {
        statusClass = 'text-green-600 font-semibold';
      } else if (log.status?.toLowerCase() === 'failure') {
        statusClass = 'text-red-600 font-semibold';
      }

      const cells = [];
      if (columnVisibility.timestamp) cells.push(<td key="timestamp" className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{getFormattedTimestamp(log.timestamp)}</td>);
      
      if (columnVisibility.tool_name) {
          cells.push(<td key="tool_name" className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{escapeHtml(log.tool_name)}</td>);
      }
      if (columnVisibility.mcp_method) {
          cells.push(<td key="mcp_method" className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{escapeHtml(log.mcp_method)}</td>);
      }

      if (columnVisibility.target_server_alias) cells.push(<td key="target_server_alias" className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{escapeHtml(log.target_server_alias)}</td>);
      if (columnVisibility.status) cells.push(<td key="status" className={`px-6 py-4 whitespace-nowrap text-sm ${statusClass}`}>{escapeHtml(log.status)}</td>);
      if (columnVisibility.duration_ms) cells.push(<td key="duration" className="px-6 py-4 whitespace-nowrap text-sm text-gray-600 text-right">{log.duration_ms !== undefined && log.duration_ms !== null ? `${log.duration_ms}ms` : '-'}</td>);
      if (columnVisibility.id) cells.push(<td key="log_id" className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{escapeHtml(log.id)}</td>);
      
      cells.push(
        <td key="details" className="px-6 py-4 whitespace-nowrap text-sm text-gray-600 text-center">
          <button
            className="view-details-btn bg-ithena-blue hover:bg-blue-600 text-white py-1 px-3 rounded-md text-xs font-medium transition duration-150 ease-in-out"
            onClick={() => onOpenModal(log.id)}
          >
            View
          </button>
        </td>
      );
      return <tr key={log.id}>{cells}</tr>;
    });
  };

  return (
    <div className="bg-white rounded-lg shadow overflow-x-auto">
      <table className="min-w-full divide-y divide-gray-200">
        <thead className="bg-gray-50">
          {renderTableHeaders()}
        </thead>
        <tbody className="bg-white divide-y divide-gray-200">
          {renderTableBody()}
        </tbody>
      </table>
    </div>
  );
}