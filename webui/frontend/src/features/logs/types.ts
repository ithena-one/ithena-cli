/* eslint-disable @typescript-eslint/no-explicit-any */

export interface LogEntry {
  id: string; 
  timestamp: string; 
  user_id?: string; 
  session_id?: string; 
  invocation_id?: string; 
  sequence?: number; 
  step_type?: string; 
  tool_name?: string | null; 
  mcp_method?: string | null; 
  mcp_host?: string | null; 
  target_server_alias?: string | null; 
  status: string; 
  duration_ms?: number | null; 
  request_payload?: any; 
  response_payload?: any; 
  error_message?: string | null; 
}

export interface LogsApiResponse {
  logs: LogEntry[];
  total_count: number;
  page: number;
  limit: number;
}

export interface FetchLogApiParams {
  page?: number;
  limit?: number;
  status?: string;
  tool_name?: string;
  mcp_method?: string;
  search?: string; 
}

export interface ColumnVisibilityState {
  timestamp: boolean;
  tool_name: boolean; 
  mcp_method: boolean;
  target_server_alias: boolean; 
  status: boolean;
  duration_ms: boolean;
  id: boolean; 
}

export const DEFAULT_COLUMN_VISIBILITY: ColumnVisibilityState = {
  timestamp: true,
  tool_name: true,
  mcp_method: true, 
  target_server_alias: true, 
  status: true,
  duration_ms: true,
  id: false, 
}; 