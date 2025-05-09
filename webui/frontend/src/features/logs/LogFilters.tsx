import React from 'react';
import { type ColumnVisibilityState } from './types';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Checkbox } from '@/components/ui/checkbox';
// import { Button } from '@/components/ui/button'; // If we add an explicit Apply button later

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

const SELECT_ALL_STATUSES_VALUE = "__all__"; // Special value for the "All Statuses" option

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
}: LogFiltersProps) {

  // const handleKeyPress = (event: React.KeyboardEvent<HTMLInputElement>) => {
  //   if (event.key === 'Enter') {
  //     // Potentially call an onApplyFilters prop if we add one
  //   }
  // };

  return (
    <div className="p-4 bg-card border border-border rounded-lg shadow-sm space-y-6"> {/* Using theme colors */}
      <div>
        <h2 className="text-lg font-semibold text-card-foreground mb-4">Filters</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 items-end">
          {/* Status Filter */}
          <div className="space-y-1.5">
            <Label htmlFor="status-filter">Status</Label>
            <Select 
              value={statusFilter === "" ? SELECT_ALL_STATUSES_VALUE : statusFilter} 
              onValueChange={(value) => {
                if (value === SELECT_ALL_STATUSES_VALUE) {
                  onStatusFilterChange("");
                } else {
                  onStatusFilterChange(value);
                }
              }}
            >
              <SelectTrigger id="status-filter" className="w-full">
                <SelectValue placeholder="All Statuses" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={SELECT_ALL_STATUSES_VALUE}>All Statuses</SelectItem>
                <SelectItem value="success">Success</SelectItem>
                <SelectItem value="failure">Failure</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Tool Name Filter */}
          <div className="space-y-1.5">
            <Label htmlFor="tool-name-filter">Tool Name</Label>
            <Input
              type="text"
              id="tool-name-filter"
              placeholder="e.g., CodebaseSearch"
              value={toolNameFilter}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => onToolNameFilterChange(e.target.value)}
              // onKeyPress={handleKeyPress} // Add if specific enter behavior needed beyond form submission
            />
          </div>

          {/* MCP Method Filter */}
          <div className="space-y-1.5">
            <Label htmlFor="mcp-method-filter">MCP Method</Label>
            <Input
              type="text"
              id="mcp-method-filter"
              placeholder="e.g., GetIssue"
              value={mcpMethodFilter}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => onMcpMethodFilterChange(e.target.value)}
              // onKeyPress={handleKeyPress}
            />
          </div>
          
          {/* Global Search Filter */}
          <div className="space-y-1.5">
            <Label htmlFor="global-search-filter">Global Search</Label>
            <Input
              type="text"
              id="global-search-filter"
              placeholder="Search all text fields..."
              value={globalSearchTerm}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => onGlobalSearchTermChange(e.target.value)}
              // onKeyPress={handleKeyPress}
            />
          </div>
           {/* Add explicit Apply Filters Button if desired for text inputs
            <Button onClick={onApplyFilters} className="md:col-start-4 lg:col-start-auto self-end">Apply</Button> 
           */}
        </div>
      </div>

      {/* Column Visibility Toggles */}
      <div className="pt-4 border-t border-border">
          <h3 className="text-md font-semibold text-card-foreground mb-3">Toggle Columns</h3>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-x-4 gap-y-3">
              {availableColumns.map((column) => (
                  <div key={column.id} className="flex items-center space-x-2">
                      <Checkbox
                          id={`col-vis-${column.id}`}
                          checked={columnVisibility[column.id]}
                          onCheckedChange={(checked: boolean | 'indeterminate') => {
                              onColumnVisibilityChange((prev) => ({
                                  ...prev,
                                  [column.id]: !!checked, // Ensure boolean value from boolean or string 'indeterminate'
                              }));
                          }}
                      />
                      <Label 
                        htmlFor={`col-vis-${column.id}`} 
                        className="text-sm font-normal text-muted-foreground cursor-pointer hover:text-foreground"
                      >
                        {column.label}
                      </Label>
                  </div>
              ))}
          </div>
      </div>
    </div>
  );
}
