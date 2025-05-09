/* eslint-disable @typescript-eslint/no-explicit-any */
import { useState, useCallback, useEffect } from 'react';
import {
  type LogEntry,
  type LogsApiResponse,
  type FetchLogApiParams,
  type ColumnVisibilityState,
  DEFAULT_COLUMN_VISIBILITY,
} from './types';

const DEFAULT_LIMIT = 20; 
export interface UseLogTableStateProps {
  initialStatusFilter?: string;
  initialToolNameFilter?: string;
  initialMcpMethodFilter?: string;
  initialGlobalSearchTerm?: string;
  initialLimit?: number;
  initialCurrentPage?: number;
  initialColumnVisibility?: Partial<ColumnVisibilityState>;
}

export function useLogTableState(props: UseLogTableStateProps = {}) {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  // Filter States
  const [statusFilter, setStatusFilter] = useState<string>(props.initialStatusFilter || '');
  const [toolNameFilter, setToolNameFilter] = useState<string>(props.initialToolNameFilter || '');
  const [mcpMethodFilter, setMcpMethodFilter] = useState<string>(props.initialMcpMethodFilter || '');
  const [globalSearchTerm, setGlobalSearchTerm] = useState<string>(props.initialGlobalSearchTerm || '');

  // Pagination States
  const [currentPage, setCurrentPage] = useState(props.initialCurrentPage || 1);
  const [limit, setLimit] = useState(props.initialLimit || DEFAULT_LIMIT);
  const totalPages = Math.ceil(totalCount / limit);

  // Column Visibility State
  const [columnVisibility, setColumnVisibility] = useState<ColumnVisibilityState>({
    ...DEFAULT_COLUMN_VISIBILITY,
    ...props.initialColumnVisibility,
  });

  // Data Fetching Logic
  const fetchData = useCallback(async (fetchParams: FetchLogApiParams) => {
    setIsLoading(true);
    setError(null);

    const queryParams = new URLSearchParams();
    if (fetchParams.page) queryParams.append('page', String(fetchParams.page));
    if (fetchParams.limit) queryParams.append('limit', String(fetchParams.limit));
    if (fetchParams.status) queryParams.append('status', fetchParams.status);
    if (fetchParams.tool_name) queryParams.append('tool_name', fetchParams.tool_name);
    if (fetchParams.mcp_method) queryParams.append('mcp_method', fetchParams.mcp_method);
    if (fetchParams.search) queryParams.append('search', fetchParams.search);

    try {
      const response = await fetch(`/api/logs?${queryParams.toString()}`);
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: response.statusText }));
        throw new Error(errorData.error || `API Error: ${response.status}`);
      }
      const data: LogsApiResponse = await response.json();
      setLogs(data.logs || []);
      console.log("Fetched logs for table:", data.logs); 
      setTotalCount(data.total_count || 0);
      setCurrentPage(data.page || 1); 
      if (data.limit) setLimit(data.limit);

    } catch (err: any) {
      console.error("Failed to fetch logs:", err);
      setError(err);
      setLogs([]); 
      setTotalCount(0);
    } finally {
      setIsLoading(false);
    }
  }, []);


  // Effect to fetch data when page, limit, or filters change
  useEffect(() => {
    fetchData({
      page: currentPage,
      limit,
      status: statusFilter,
      tool_name: toolNameFilter,
      mcp_method: mcpMethodFilter,
      search: globalSearchTerm,
    });
  }, [currentPage, limit, statusFilter, toolNameFilter, mcpMethodFilter, globalSearchTerm, fetchData]);

  // Handlers
  const handlePageChange = (newPage: number) => {
    if (newPage > 0 && newPage <= totalPages) {
      setCurrentPage(newPage);
    }
  };

  const handleLimitChange = (newLimit: number) => {
    setLimit(newLimit);
    setCurrentPage(1); // Reset to page 1 when limit changes
  };
  
  const applyFilters = (filters: {
    status?: string,
    toolName?: string,
    mcpMethod?: string,
    search?: string,
  }) => {
    setStatusFilter(filters.status ?? statusFilter);
    setToolNameFilter(filters.toolName ?? toolNameFilter);
    setMcpMethodFilter(filters.mcpMethod ?? mcpMethodFilter);
    setGlobalSearchTerm(filters.search ?? globalSearchTerm);
    setCurrentPage(1); // Reset to page 1 when filters are applied
     // Data will refetch due to useEffect dependencies
  };


  // Expose state and handlers
  return {
    logs,
    totalCount,
    currentPage,
    totalPages,
    limit,
    isLoading,
    error,
    
    statusFilter,
    toolNameFilter,
    mcpMethodFilter,
    globalSearchTerm,
    columnVisibility,

    // Handlers/Setters
    handlePageChange,
    handleLimitChange,
    // Individual filter setters can be exposed if needed, or use applyFilters
    setStatusFilter: (status: string) => { setStatusFilter(status); setCurrentPage(1); },
    setToolNameFilter: (name: string) => { setToolNameFilter(name); setCurrentPage(1); },
    setMcpMethodFilter: (method: string) => { setMcpMethodFilter(method); setCurrentPage(1); },
    setGlobalSearchTerm: (term: string) => { setGlobalSearchTerm(term); setCurrentPage(1); },
    applyFilters, // More comprehensive filter update
    setColumnVisibility,
    refreshData: () => fetchData({ // Exposed refresh function
        page: currentPage,
        limit,
        status: statusFilter,
        tool_name: toolNameFilter,
        mcp_method: mcpMethodFilter,
        search: globalSearchTerm,
    }),
  };
}