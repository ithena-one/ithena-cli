document.addEventListener('DOMContentLoaded', () => {
    const logTableBody = document.getElementById('log-table-body');
    const logDetailModal = document.getElementById('log-detail-modal');
    const logDetailContent = document.getElementById('log-detail-content');
    const closeModalButton = logDetailModal.querySelector('.close-button');

    // Pagination state and elements
    let currentPage = 1;
    const logsPerPage = 20; // Align with backend default or make configurable if needed
    const paginationContainer = document.getElementById('pagination');
    
    // Filter state and elements
    const filterStatusInput = document.createElement('select');
    filterStatusInput.className = 'p-2 border border-gray-300 rounded-md shadow-sm focus:ring-ithena-blue focus:border-ithena-blue sm:text-sm';
    filterStatusInput.innerHTML = `<option value="">All Statuses</option><option value="success">Success</option><option value="failure">Failure</option>`;
    
    const filterToolNameInput = document.createElement('input');
    filterToolNameInput.type = 'text';
    filterToolNameInput.placeholder = 'Filter by Tool Name...';
    filterToolNameInput.className = 'p-2 border border-gray-300 rounded-md shadow-sm focus:ring-ithena-blue focus:border-ithena-blue sm:text-sm w-full md:w-auto';

    const filterSearchInput = document.createElement('input');
    filterSearchInput.type = 'text';
    filterSearchInput.placeholder = 'Global Search...';
    filterSearchInput.className = 'p-2 border border-gray-300 rounded-md shadow-sm focus:ring-ithena-blue focus:border-ithena-blue sm:text-sm w-full md:w-auto';

    const applyFiltersButton = document.createElement('button');
    applyFiltersButton.textContent = 'Apply Filters';
    applyFiltersButton.className = 'ml-2 py-2 px-4 bg-ithena-blue hover:bg-blue-700 text-white font-semibold rounded-md shadow-sm transition duration-150 ease-in-out';

    const filtersDiv = document.getElementById('filters');
    filtersDiv.innerHTML = ''; // Clear placeholder
    filtersDiv.className = 'flex flex-wrap items-center gap-4 mb-6 p-4 bg-white rounded-lg shadow'; // Use flex for layout

    const statusLabel = document.createElement('label');
    statusLabel.textContent = 'Status:';
    statusLabel.className = 'font-medium text-sm text-gray-700';
    filtersDiv.appendChild(statusLabel);
    filtersDiv.appendChild(filterStatusInput);

    const toolLabel = document.createElement('label');
    toolLabel.textContent = 'Tool:';
    toolLabel.className = 'font-medium text-sm text-gray-700';
    filtersDiv.appendChild(toolLabel);
    filtersDiv.appendChild(filterToolNameInput);

    const searchLabel = document.createElement('label');
    searchLabel.textContent = 'Search:';
    searchLabel.className = 'font-medium text-sm text-gray-700';
    filtersDiv.appendChild(searchLabel);
    filtersDiv.appendChild(filterSearchInput);
    
    filtersDiv.appendChild(applyFiltersButton);

    const authStatusEl = document.getElementById('auth-status');
    const authActionButton = document.getElementById('auth-action-button');
    const platformLink = document.getElementById('platform-link'); // Though its href is static, we might change text or behavior later.

    const ITHENA_PLATFORM_URL = 'https://ithena.one';
    platformLink.href = ITHENA_PLATFORM_URL;

    let totalLogs = 0;

    async function fetchAuthStatus() {
        try {
            const response = await fetch('/api/auth/status');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            const data = await response.json();
            updateAuthUI(data);
        } catch (error) {
            console.error('Failed to fetch auth status:', error);
            authStatusEl.textContent = 'Error loading auth status.';
            authActionButton.textContent = 'Login';
            authActionButton.onclick = () => {
                alert('Could not determine auth status. Please ensure the Ithena CLI server is running correctly.');
            };
        }
    }

    function updateAuthUI(authData) {
        if (authData.authenticated) {
            authStatusEl.textContent = 'Status: Authenticated';
            authActionButton.textContent = 'View My Account on Ithena';
            authActionButton.onclick = () => {
                window.open(ITHENA_PLATFORM_URL, '_blank');
            };
        } else {
            authStatusEl.textContent = 'Status: Unauthenticated';
            authActionButton.textContent = 'Login with Ithena';
            authActionButton.onclick = () => {
                alert('To log in, please run `ithena-cli auth login` in your terminal.');
            };
        }
    }

    async function fetchAndDisplayLogs(page = 1) {
        currentPage = page;
        logTableBody.innerHTML = `<tr><td colspan="5">Loading logs (page ${page})...</td></tr>`;

        const statusFilter = filterStatusInput.value;
        const toolNameFilter = filterToolNameInput.value.trim();
        const searchTerm = filterSearchInput.value.trim();

        let apiUrl = `/api/logs?page=${page}&limit=${logsPerPage}`;
        if (statusFilter) apiUrl += `&status=${encodeURIComponent(statusFilter)}`;
        if (toolNameFilter) apiUrl += `&tool_name=${encodeURIComponent(toolNameFilter)}`;
        if (searchTerm) apiUrl += `&search=${encodeURIComponent(searchTerm)}`;

        try {
            const response = await fetch(apiUrl);
            if (!response.ok) {
                logTableBody.innerHTML = `<tr><td colspan="5">Error loading logs: ${response.statusText}</td></tr>`;
                console.error("API Error:", response);
                updatePagination(0, page, logsPerPage); // Show no pages
                return;
            }
            const data = await response.json();
            
            if (!data.logs || data.logs.length === 0) {
                logTableBody.innerHTML = '<tr><td colspan="5">No logs found.</td></tr>';
                updatePagination(0, page, logsPerPage);
                return;
            }

            logTableBody.innerHTML = ''; // Clear loading message
            data.logs.forEach(log => {
                const row = document.createElement('tr');
                row.dataset.logId = log.id; // Store ID for click events

                const timestamp = new Date(log.timestamp).toLocaleString();
                const toolOrMethod = log.tool_name || log.mcp_method || '-';
                const duration = log.duration_ms !== null ? `${log.duration_ms}ms` : '-';
                
                // Determine status color class - prioritize Tailwind if migrating
                let statusClass = 'text-gray-700'; // Default
                if (log.status.toLowerCase() === 'success') {
                    statusClass = 'text-green-600 font-semibold';
                } else if (log.status.toLowerCase() === 'failure') {
                    statusClass = 'text-red-600 font-semibold';
                }

                row.innerHTML = `
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-600">${timestamp}</td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-600">${escapeHtml(toolOrMethod)}</td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm ${statusClass}">${escapeHtml(log.status)}</td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-600 text-right">${duration}</td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-600 text-center">
                        <button class="view-details-btn bg-ithena-blue hover:bg-blue-600 text-white py-1 px-3 rounded-md text-xs font-medium transition duration-150 ease-in-out" data-log-id="${log.id}">View</button>
                    </td>
                `;
                logTableBody.appendChild(row);
            });
            updatePagination(data.total_count, data.page, data.limit);
        } catch (error) {
            logTableBody.innerHTML = `<tr><td colspan="5">Failed to fetch logs: ${error.message}</td></tr>`;
            console.error("Fetch Error:", error);
            updatePagination(0, page, logsPerPage);
        }
    }

    function escapeHtml(unsafe) {
        if (unsafe === null || typeof unsafe === 'undefined') return '-';
        if (typeof unsafe !== 'string') return unsafe.toString();
        return unsafe
             .replace(/&/g, "&amp;")
             .replace(/</g, "&lt;")
             .replace(/>/g, "&gt;")
             .replace(/"/g, "&quot;")
             .replace(/'/g, "&#039;");
     }

    function updatePagination(totalLogs, currentPage, logsPerPage) {
        paginationContainer.innerHTML = ''; // Clear previous pagination
        if (totalLogs === 0) {
            paginationContainer.innerHTML = '<p class="text-sm text-gray-500">No logs to paginate.</p>';
            return;
        }

        const totalPages = Math.ceil(totalLogs / logsPerPage);

        if (totalPages <= 1 && totalLogs > 0) { 
            paginationContainer.innerHTML = `<p class="text-sm text-gray-500">Page 1 of 1 (${totalLogs} logs)</p>`;
            return; 
        }
        if (totalPages <=1) return;

        const prevButton = document.createElement('button');
        prevButton.textContent = 'Previous';
        prevButton.className = 'py-2 px-4 border border-gray-300 bg-white text-sm font-medium rounded-md text-gray-700 hover:bg-gray-50 disabled:opacity-50';
        prevButton.disabled = currentPage === 1;
        prevButton.addEventListener('click', () => fetchAndDisplayLogs(currentPage - 1));
        paginationContainer.appendChild(prevButton);

        const pageInfo = document.createElement('span');
        pageInfo.textContent = ` Page ${currentPage} of ${totalPages} `;
        pageInfo.className = "text-sm text-gray-700 mx-2";
        paginationContainer.appendChild(pageInfo);

        const nextButton = document.createElement('button');
        nextButton.textContent = 'Next';
        nextButton.className = 'py-2 px-4 border border-gray-300 bg-white text-sm font-medium rounded-md text-gray-700 hover:bg-gray-50 disabled:opacity-50';
        nextButton.disabled = currentPage === totalPages;
        nextButton.addEventListener('click', () => fetchAndDisplayLogs(currentPage + 1));
        paginationContainer.appendChild(nextButton);
    }

    async function showLogDetails(logId) {
        logDetailContent.textContent = 'Loading details...';
        logDetailModal.style.display = 'block';
        try {
            const response = await fetch(`/api/logs/${logId}`);
            if (!response.ok) {
                logDetailContent.textContent = `Error loading details: ${response.statusText}`;
                return;
            }
            const logData = await response.json();
            logDetailContent.textContent = JSON.stringify(logData, null, 2);
        } catch (error) {
            logDetailContent.textContent = `Failed to fetch details: ${error.message}`;
        }
    }

    logTableBody.addEventListener('click', (event) => {
        const targetButton = event.target.closest('.view-details-btn');
        if (targetButton) {
            const logId = targetButton.dataset.logId;
            showLogDetails(logId);
        }
    });

    closeModalButton.addEventListener('click', () => {
        logDetailModal.style.display = 'none';
    });

    window.addEventListener('click', (event) => {
        if (event.target === logDetailModal) {
            logDetailModal.style.display = 'none';
        }
    });

    applyFiltersButton.addEventListener('click', () => {
        fetchAndDisplayLogs(1); // Fetch from page 1 with new filters
    });
    filterSearchInput.addEventListener('keypress', (event) => {
        if (event.key === 'Enter') {
            fetchAndDisplayLogs(1);
        }
    });
     filterToolNameInput.addEventListener('keypress', (event) => {
        if (event.key === 'Enter') {
            fetchAndDisplayLogs(1);
        }
    });

    // Initial load
    fetchAuthStatus();
    fetchAndDisplayLogs();
}); 