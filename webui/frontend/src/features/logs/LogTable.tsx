'use client'; // If using Next.js App Router, though not strictly needed for plain Vite React

import { useState, useCallback } from 'react';
import { useLogTableState, type UseLogTableStateProps } from './useLogTableState';
import LogFilters from './LogFilters';
import LogTableContent from './LogTableContent';
import LogPaginationControls from './LogPaginationControl';
import JsonDetailModal from './JsonDetailModal';
import { DEFAULT_COLUMN_VISIBILITY, type ColumnVisibilityState } from './types';
import Header from './Header';


export type LogTableContainerProps = UseLogTableStateProps

const toLabel = (key: string) => {
  const result = key.replace(/([A-Z])|_ms/g, (_match, p1) => p1 ? ` ${p1}` : '').replace(/_/g, ' ');
  const finalResult = result.charAt(0).toUpperCase() + result.slice(1);
  return finalResult.endsWith(' Id') ? finalResult.replace(' Id', ' ID') : finalResult;
};


export default function LogTable(props: LogTableContainerProps) {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [selectedLogId, setSelectedLogId] = useState<string | null>(null);

  const {
    logs,
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
    handlePageChange,
    handleLimitChange, // Added from hook
    setStatusFilter,
    setToolNameFilter,
    setMcpMethodFilter,
    setGlobalSearchTerm,
    setColumnVisibility,
    // applyFilters, // Can be used if LogFilters has an explicit apply button
    // refreshData // Can be used for a manual refresh button
  } = useLogTableState(props);

  const openModal = useCallback((logId: string) => {
    setSelectedLogId(logId);
    setIsModalOpen(true);
  }, []);

  const closeModal = useCallback(() => {
    setIsModalOpen(false);
    setSelectedLogId(null);
  }, []);
  
 
  const availableFilterColumns = Object.keys(DEFAULT_COLUMN_VISIBILITY).map(key => ({
    id: key as keyof ColumnVisibilityState,
    label: toLabel(key), 
  }));


  return (
    <div className="space-y-6 p-4 sm:p-6 lg:p-8 bg-gray-50 min-h-screen">
      <Header />
      
      {error && (
        <div className="p-4 mb-4 text-sm text-red-700 bg-red-100 rounded-lg" role="alert">
          <span className="font-medium">Error:</span> {error.message}
        </div>
      )}

      <LogFilters
        statusFilter={statusFilter}
        onStatusFilterChange={setStatusFilter}
        toolNameFilter={toolNameFilter}
        onToolNameFilterChange={setToolNameFilter}
        mcpMethodFilter={mcpMethodFilter}
        onMcpMethodFilterChange={setMcpMethodFilter}
        globalSearchTerm={globalSearchTerm}
        onGlobalSearchTermChange={setGlobalSearchTerm}
        columnVisibility={columnVisibility}
        onColumnVisibilityChange={setColumnVisibility}
        availableColumns={availableFilterColumns}
      />

      <LogTableContent
        logs={logs}
        isLoading={isLoading}
        // limit={limit} // Pass limit if LogTableContent uses it
        columnVisibility={columnVisibility}
        onOpenModal={openModal}
        error={error} // Pass error to table content as well for specific rendering there
      />

      {totalPages > 0 && (
        <LogPaginationControls
          currentPage={currentPage}
          totalPages={totalPages}
          isLoading={isLoading}
          onPageChange={handlePageChange}
        />
      )}
      
      <div className="flex justify-end items-center text-sm mt-4">
        <label htmlFor="items-per-page" className="mr-2 text-gray-700">Items per page:</label>
        <select
          id="items-per-page"
          value={limit}
          onChange={(e) => handleLimitChange(Number(e.target.value))}
          className="p-2 border border-gray-300 rounded-md shadow-sm focus:ring-ithena-blue focus:border-ithena-blue sm:text-sm"
          disabled={isLoading}
        >
          <option value={10}>10</option>
          <option value={20}>20</option>
          <option value={50}>50</option>
          <option value={100}>100</option>
        </select>
      </div>


      <JsonDetailModal isOpen={isModalOpen} onClose={closeModal} logId={selectedLogId} />
    </div>
  );
} 