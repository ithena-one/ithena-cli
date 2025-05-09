import React from 'react';
import { type ColumnVisibilityState } from './types';

interface LogFiltersProps {
  statusFilter: string;
  onStatusFilterChange: (value: string) => void;
  toolNameFilter: string;
  onToolNameFilterChange: (value: string) => void;
  mcpMethodFilter: string;
  onMcpMethodFilterChange: (value: string) => void;
  globalSearchTerm: string;
  onGlobalSearchTermChange: (value: string) => void;
  
  columnVisibility: ColumnVisibilityState;
  onColumnVisibilityChange: (updater: (prev: ColumnVisibilityState) => ColumnVisibilityState) => void;

  availableColumns: Array<{ id: keyof ColumnVisibilityState; label: string }>;

}


export default function LogFilters({
  statusFilter,
  onStatusFilterChange,
  toolNameFilter,
  onToolNameFilterChange,
  mcpMethodFilter,
  onMcpMethodFilterChange,
  globalSearchTerm,
  onGlobalSearchTermChange,
  columnVisibility,
  onColumnVisibilityChange,
  availableColumns,
  // onApplyFilters
}: LogFiltersProps) {

  const handleKeyPress = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter') {
      // If we had an explicit onApplyFilters, we might call it here.
      // Currently, changes to inputs directly update state in useLogTableState,
      // which triggers a refetch via useEffect.
    }
  };

  return (
    <div className="p-4 bg-white rounded-lg shadow-md space-y-4">
      <h2 className="text-lg font-semibold text-gray-700">Filters</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Status Filter */}
        <div>
          <label htmlFor="status-filter" className="block text-sm font-medium text-gray-700">
            Status
          </label>
          <select
            id="status-filter"
            name="status"
            className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-ithena-blue focus:border-ithena-blue sm:text-sm rounded-md"
            value={statusFilter}
            onChange={(e) => onStatusFilterChange(e.target.value)}
          >
            <option value="">All</option>
            <option value="success">Success</option>
            <option value="failure">Failure</option>
            {/* Add other statuses if relevant */}
          </select>
        </div>

        {/* Tool Name Filter */}
        <div>
          <label htmlFor="tool-name-filter" className="block text-sm font-medium text-gray-700">
            Tool Name
          </label>
          <input
            type="text"
            id="tool-name-filter"
            name="toolName"
            className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-ithena-blue focus:border-ithena-blue sm:text-sm"
            placeholder="e.g., CodebaseSearch"
            value={toolNameFilter}
            onChange={(e) => onToolNameFilterChange(e.target.value)}
            onKeyPress={handleKeyPress}
          />
        </div>

        {/* MCP Method Filter */}
        <div>
          <label htmlFor="mcp-method-filter" className="block text-sm font-medium text-gray-700">
            MCP Method
          </label>
          <input
            type="text"
            id="mcp-method-filter"
            name="mcpMethod"
            className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-ithena-blue focus:border-ithena-blue sm:text-sm"
            placeholder="e.g., GetIssue"
            value={mcpMethodFilter}
            onChange={(e) => onMcpMethodFilterChange(e.target.value)}
            onKeyPress={handleKeyPress}
          />
        </div>
        
        {/* Global Search Filter */}
        <div>
          <label htmlFor="global-search-filter" className="block text-sm font-medium text-gray-700">
            Global Search
          </label>
          <input
            type="text"
            id="global-search-filter"
            name="globalSearch"
            className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-ithena-blue focus:border-ithena-blue sm:text-sm"
            placeholder="Search all text fields..."
            value={globalSearchTerm}
            onChange={(e) => onGlobalSearchTermChange(e.target.value)}
            onKeyPress={handleKeyPress}
          />
        </div>
      </div>

      {/* Column Visibility Toggles */}
      <div className="pt-4">
          <h3 className="text-md font-semibold text-gray-700 mb-2">Toggle Columns</h3>
          <div className="flex flex-wrap gap-x-4 gap-y-2">
              {availableColumns.map((column) => (
                  <label key={column.id} className="inline-flex items-center space-x-2 cursor-pointer">
                      <input
                          type="checkbox"
                          className="form-checkbox h-4 w-4 text-ithena-blue border-gray-300 rounded focus:ring-ithena-blue"
                          checked={columnVisibility[column.id]}
                          onChange={() => {
                              onColumnVisibilityChange((prev) => ({
                                  ...prev,
                                  [column.id]: !prev[column.id],
                              }));
                          }}
                      />
                      <span className="text-sm text-gray-600">{column.label}</span>
                  </label>
              ))}
          </div>
      </div>
      

    </div>
  );
}
