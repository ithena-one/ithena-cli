/* eslint-disable @typescript-eslint/no-explicit-any */
import { useState, useEffect } from 'react';
import ReactJson from 'react-json-view';
import { type LogEntry } from './types'; 

interface JsonDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  logId: string | null;
}

export default function JsonDetailModal({ isOpen, onClose, logId }: JsonDetailModalProps) {
  const [logDetail, setLogDetail] = useState<LogEntry | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    if (isOpen && logId) {
      const fetchLogDetails = async () => {
        setIsLoading(true);
        setError(null);
        setLogDetail(null); 
        try {
          const response = await fetch(`/api/logs/${logId}`);
          if (!response.ok) {
            const errorData = await response.json().catch(() => ({ message: response.statusText }));
            throw new Error(errorData.error || `API Error: ${response.status}`);
          }
          const data: LogEntry = await response.json();
          setLogDetail(data);
        } catch (err: any) {
          console.error(`Failed to fetch details for log ${logId}:`, err);
          setError(err);
        } finally {
          setIsLoading(false);
        }
      };
      fetchLogDetails();
    } else if (!isOpen) {
      setLogDetail(null);
      setError(null);
    }
  }, [isOpen, logId]);

  if (!isOpen) {
    return null;
  }

  return (
    <div
      id="log-detail-modal"
      className="fixed inset-0 bg-gray-600 bg-opacity-75 overflow-y-auto h-full w-full flex items-center justify-center z-50 transition-opacity duration-300 ease-in-out"
      onClick={onClose} 
    >
      <div
        className="modal-content bg-white p-5 sm:p-6 rounded-lg shadow-xl w-11/12 md:w-3/4 lg:w-2/3 max-h-[85vh] overflow-y-hidden flex flex-col transform transition-all duration-300 ease-in-out scale-95 opacity-0 animate-modalShow"
        onClick={(e) => e.stopPropagation()} 
        role="dialog"
        aria-modal="true"
        aria-labelledby="modal-title"
      >
        <div className="flex justify-between items-center mb-4 pb-3 border-b border-gray-200">
          <h2 id="modal-title" className="text-xl sm:text-2xl font-semibold text-ithena-gray">
            Log Details {logId && <span className="text-sm text-gray-500">(ID: {logId})</span>}
          </h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors p-1 rounded-full focus:outline-none focus:ring-2 focus:ring-ithena-blue"
            aria-label="Close modal"
          >
            <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="overflow-y-auto flex-grow pr-1 scrollbar-thin scrollbar-thumb-gray-400 scrollbar-track-gray-200 scrollbar-thumb-rounded">
          {isLoading && <p className="text-gray-700 text-center py-4">Loading details...</p>}
          {error && <p className="text-red-600 text-center py-4">Error loading details: {error.message}</p>}
          {logDetail && !isLoading && !error && (
            <div className="bg-gray-50 p-3 sm:p-4 rounded text-xs sm:text-sm">
              <ReactJson
                src={logDetail as object}
                name={null}
                theme="rjv-default"
                collapsed={1}
                enableClipboard={copy => {
                  navigator.clipboard.writeText(JSON.stringify(copy.src, null, 2));
                  return true;
                }}
                displayObjectSize={true}
                displayDataTypes={true}
                quotesOnKeys={false}
                style={{ padding: '1em', backgroundColor: 'transparent' }}
              />
            </div>
          )}
          {!logDetail && !isLoading && !error && <p className="text-gray-500 text-center py-4">No details available.</p>}
        </div>
      </div>
    </div>
  );
} 