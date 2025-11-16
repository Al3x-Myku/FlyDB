const API_BASE = 'http://localhost:8080';

let currentPage = 'dashboard';
let currentCollection = null;
let collections = [];
let stats = {};
let compressionEnabled = true;
let editingDocument = null;

document.addEventListener('DOMContentLoaded', () => {
    loadDashboard();
    loadCompressionSetting();
    setInterval(() => refreshData(), 30000);
});

function toggleTheme() {
    document.documentElement.classList.toggle('dark');
}


function showPage(page) {
    currentPage = page;
    switch(page) {
        case 'dashboard':
            loadDashboard();
            break;
        case 'collections':
            loadCollectionsPage();
            break;
        case 'query':
            loadQueryPage();
            break;
        case 'settings':
            loadSettingsPage();
            break;
    }
    updateActiveNav(page);
}

function updateActiveNav(page) {
    document.querySelectorAll('nav a').forEach(link => {
        link.classList.remove('bg-blue-light', 'dark:bg-primary/20', 'text-blue-strong', 'dark:text-white');
        link.classList.add('text-gray-light', 'dark:text-gray-400');
    });
    event.target.closest('a').classList.add('bg-blue-light', 'dark:bg-primary/20', 'text-blue-strong', 'dark:text-white');
    event.target.closest('a').classList.remove('text-gray-light', 'dark:text-gray-400');
}

async function loadDashboard() {
    currentPage = 'dashboard';
    try {
        const [statsData, collectionsData] = await Promise.all([
            fetchAPI('/api/stats'),
            fetchAPI('/api/collections')
        ]);
        
        stats = statsData;
        collections = collectionsData.collections || [];
        
        document.getElementById('db-path').textContent = stats.data_dir;
        renderDashboard();
    } catch (error) {
        showError('Failed to load dashboard data: ' + error.message);
    }
}

async function refreshData() {
    switch(currentPage) {
        case 'dashboard':
            await loadDashboard();
            break;
        case 'collections':
            if (currentCollection) {
                await viewCollection(currentCollection);
            } else {
                await loadCollectionsPage();
            }
            break;
        case 'query':
            const data = await fetchAPI('/api/collections');
            collections = data.collections || [];
            loadQueryPage();
            break;
        case 'settings':
            await loadSettingsPage();
            break;
    }
    await loadCompressionSetting();
    showNotification('Data refreshed');
}

async function loadCompressionSetting() {
    try {
        const data = await fetchAPI('/api/settings/compression');
        compressionEnabled = data.compression;
    } catch (error) {
        console.error('Failed to load compression setting:', error);
    }
}

function renderDashboard() {
    const totalDocs = Object.values(stats.collections || {}).reduce((sum, col) => sum + col.total_size, 0);
    const totalMemtable = Object.values(stats.collections || {}).reduce((sum, col) => sum + col.memtable_size, 0);
    const totalIndexed = Object.values(stats.collections || {}).reduce((sum, col) => sum + col.index_size, 0);
    
    const content = `
        <div class="flex flex-wrap items-center justify-between gap-4">
            <div class="flex flex-col gap-1">
                <p class="text-gray-dark dark:text-white text-3xl font-black leading-tight">Dashboard</p>
                <p class="text-gray-light dark:text-gray-400 text-base font-normal leading-normal">Overview of your FlyDB database health and performance.</p>
            </div>
            <div class="flex items-center gap-2">
                <span class="text-sm text-gray-light dark:text-gray-400">Last updated: <span id="last-updated">just now</span></span>
                <button onclick="refreshData()" class="flex h-10 min-w-[84px] cursor-pointer items-center justify-center gap-2 overflow-hidden rounded-lg bg-white dark:bg-gray-800 px-4 text-gray-dark dark:text-white text-sm font-medium leading-normal border border-gray-200 dark:border-gray-700 hover:bg-gray-100 dark:hover:bg-gray-700">
                    <span class="material-symbols-outlined text-lg">refresh</span>
                    <span class="truncate">Refresh</span>
                </button>
            </div>
        </div>
        
        <div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
            <div class="flex flex-col gap-2 rounded-xl p-6 bg-white dark:bg-[#111a22] border border-gray-200 dark:border-gray-800">
                <p class="text-gray-light dark:text-gray-400 text-base font-medium leading-normal">DB Health Status</p>
                <div class="flex items-center gap-2">
                    <span class="w-3 h-3 rounded-full bg-green-status"></span>
                    <p class="text-gray-dark dark:text-white tracking-light text-2xl font-bold leading-tight">Healthy</p>
                </div>
            </div>
            <div class="flex flex-col gap-2 rounded-xl p-6 bg-white dark:bg-[#111a22] border border-gray-200 dark:border-gray-800">
                <p class="text-gray-light dark:text-gray-400 text-base font-medium leading-normal">Collections</p>
                <div class="flex items-baseline gap-2">
                    <p class="text-gray-dark dark:text-white tracking-light text-2xl font-bold leading-tight">${stats.collections_count || 0}</p>
                </div>
            </div>
            <div class="flex flex-col gap-2 rounded-xl p-6 bg-white dark:bg-[#111a22] border border-gray-200 dark:border-gray-800">
                <p class="text-gray-light dark:text-gray-400 text-base font-medium leading-normal">Total Documents</p>
                <div class="flex items-baseline gap-2">
                    <p class="text-gray-dark dark:text-white tracking-light text-2xl font-bold leading-tight">${totalDocs.toLocaleString()}</p>
                </div>
            </div>
            <div class="flex flex-col gap-2 rounded-xl p-6 bg-white dark:bg-[#111a22] border border-gray-200 dark:border-gray-800">
                <p class="text-gray-light dark:text-gray-400 text-base font-medium leading-normal">Uncommitted Docs</p>
                <p class="text-gray-dark dark:text-white tracking-light text-2xl font-bold leading-tight">${totalMemtable.toLocaleString()}</p>
            </div>
        </div>
        
        <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
            <div class="flex flex-col gap-4 rounded-xl border border-gray-200 dark:border-gray-800 p-6 bg-white dark:bg-[#111a22] lg:col-span-2">
                <h3 class="text-gray-dark dark:text-white text-lg font-bold leading-tight">Collections Overview</h3>
                <div class="overflow-x-auto">
                    <table class="w-full text-left text-sm">
                        <thead class="border-b border-gray-200 dark:border-gray-800 text-gray-light dark:text-gray-400">
                            <tr>
                                <th class="py-2 px-3 font-medium">Collection</th>
                                <th class="py-2 px-3 font-medium">Memtable</th>
                                <th class="py-2 px-3 font-medium">Indexed</th>
                                <th class="py-2 px-3 font-medium">Total</th>
                                <th class="py-2 px-3 font-medium">Actions</th>
                            </tr>
                        </thead>
                        <tbody class="text-gray-dark dark:text-gray-300">
                            ${renderCollectionsTable()}
                        </tbody>
                    </table>
                </div>
            </div>
            
            <!-- Quick Actions -->
            <div class="flex flex-col gap-4 rounded-xl border border-gray-200 dark:border-gray-800 p-6 bg-white dark:bg-[#111a22]">
                <h3 class="text-gray-dark dark:text-white text-lg font-bold leading-tight">Quick Actions</h3>
                <div class="flex flex-col gap-3">
                    <button onclick="showPage('collections')" class="flex h-10 w-full cursor-pointer items-center justify-center overflow-hidden rounded-lg bg-blue-strong text-white text-sm font-medium leading-normal hover:bg-blue-strong/90">
                        Browse Collections
                    </button>
                    <button onclick="showPage('query')" class="flex h-10 w-full cursor-pointer items-center justify-center overflow-hidden rounded-lg border border-blue-strong text-blue-strong text-sm font-medium leading-normal hover:bg-blue-light dark:hover:bg-blue-strong/20">
                        Query Editor
                    </button>
                    <button onclick="commitAllCollections()" class="flex h-10 w-full cursor-pointer items-center justify-center overflow-hidden rounded-lg border border-gray-300 dark:border-gray-700 text-gray-dark dark:text-white text-sm font-medium leading-normal hover:bg-gray-100 dark:hover:bg-gray-800">
                        Commit All
                    </button>
                </div>
            </div>
        </div>
        
        <!-- Database Info -->
        <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
            <div class="flex flex-col gap-4 rounded-xl border border-gray-200 dark:border-gray-800 p-6 bg-white dark:bg-[#111a22] lg:col-span-1">
                <h3 class="text-gray-dark dark:text-white text-lg font-bold leading-tight">Database Information</h3>
                <ul class="space-y-3 text-sm">
                    <li class="flex justify-between">
                        <span class="text-gray-light dark:text-gray-400">Database Engine:</span>
                        <span class="font-medium text-gray-dark dark:text-white">FlyDB v1.0</span>
                    </li>
                    <li class="flex justify-between">
                        <span class="text-gray-light dark:text-gray-400">Storage Format:</span>
                        <span class="font-medium text-gray-dark dark:text-white">TOON + Gzip</span>
                    </li>
                    <li class="flex justify-between">
                        <span class="text-gray-light dark:text-gray-400">Data Directory:</span>
                        <span class="font-medium text-gray-dark dark:text-white truncate ml-2" title="${stats.data_dir}">${stats.data_dir?.split('/').pop() || 'N/A'}</span>
                    </li>
                    <li class="flex justify-between">
                        <span class="text-gray-light dark:text-gray-400">Architecture:</span>
                        <span class="font-medium text-gray-dark dark:text-white">Memtable + LSM</span>
                    </li>
                </ul>
            </div>
            
            <!-- Recent Activity -->
            <div class="flex flex-col gap-4 rounded-xl border border-gray-200 dark:border-gray-800 p-6 bg-white dark:bg-[#111a22] lg:col-span-2">
                <h3 class="text-gray-dark dark:text-white text-lg font-bold leading-tight">Collection Statistics</h3>
                <div class="space-y-3">
                    ${renderCollectionStats()}
                </div>
            </div>
        </div>
    `;
    
    document.getElementById('main-content').innerHTML = content;
}

function renderCollectionsTable() {
    if (!stats.collections || Object.keys(stats.collections).length === 0) {
        return '<tr><td colspan="5" class="py-4 px-3 text-center text-gray-light dark:text-gray-400">No collections found</td></tr>';
    }
    
    return Object.entries(stats.collections).map(([name, col]) => `
        <tr class="border-b border-gray-200 dark:border-gray-800">
            <td class="py-3 px-3 font-medium">${name}</td>
            <td class="py-3 px-3">${col.memtable_size}</td>
            <td class="py-3 px-3">${col.index_size}</td>
            <td class="py-3 px-3">${col.total_size}</td>
            <td class="py-3 px-3">
                <button onclick="viewCollection('${name}')" class="text-blue-strong hover:underline">View</button>
            </td>
        </tr>
    `).join('');
}

function renderCollectionStats() {
    if (!stats.collections || Object.keys(stats.collections).length === 0) {
        return '<p class="text-gray-light dark:text-gray-400 text-sm">No collections to display</p>';
    }
    
    return Object.entries(stats.collections).map(([name, col]) => {
        const percentage = col.total_size > 0 ? ((col.memtable_size / col.total_size) * 100).toFixed(1) : 0;
        return `
            <div class="flex flex-col gap-2">
                <div class="flex justify-between items-center">
                    <span class="text-sm font-medium text-gray-dark dark:text-white">${name}</span>
                    <span class="text-sm text-gray-light dark:text-gray-400">${col.total_size} docs</span>
                </div>
                <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                    <div class="bg-blue-strong h-2 rounded-full" style="width: ${percentage}%"></div>
                </div>
                <div class="flex justify-between text-xs text-gray-light dark:text-gray-400">
                    <span>Committed: ${col.index_size}</span>
                    <span>Uncommitted: ${col.memtable_size}</span>
                </div>
            </div>
        `;
    }).join('');
}

async function loadCollectionsPage() {
    currentPage = 'collections';
    currentCollection = null;
    try {
        const data = await fetchAPI('/api/collections');
        collections = data.collections || [];
        renderCollectionsPage();
    } catch (error) {
        showError('Failed to load collections: ' + error.message);
    }
}

function renderCollectionsPage() {
    const content = `
        <div class="flex flex-wrap items-center justify-between gap-4">
            <div class="flex flex-col gap-1">
                <p class="text-gray-dark dark:text-white text-3xl font-black leading-tight">Collections</p>
                <p class="text-gray-light dark:text-gray-400 text-base font-normal leading-normal">Browse and manage your collections.</p>
            </div>
            <button onclick="showCreateCollectionModal()" class="flex h-10 px-4 items-center justify-center gap-2 rounded-lg bg-green-status text-white text-sm font-medium hover:bg-green-status/90">
                <span class="material-symbols-outlined">add</span>
                Create Collection
            </button>
        </div>
        
        <div class="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
            ${collections.map(name => renderCollectionCard(name)).join('')}
        </div>
    `;
    
    document.getElementById('main-content').innerHTML = content;
}

function renderCollectionCard(name) {
    const col = stats.collections?.[name] || { memtable_size: 0, index_size: 0, total_size: 0 };
    return `
        <div class="flex flex-col gap-4 rounded-xl border border-gray-200 dark:border-gray-800 p-6 bg-white dark:bg-[#111a22]">
            <div class="flex items-center justify-between">
                <h3 class="text-gray-dark dark:text-white text-lg font-bold">${name}</h3>
                <button onclick="deleteCollection('${name}')" class="text-red-status hover:text-red-600">
                    <span class="material-symbols-outlined">delete</span>
                </button>
            </div>
            <div class="space-y-2 text-sm">
                <div class="flex justify-between">
                    <span class="text-gray-light dark:text-gray-400">Total Documents:</span>
                    <span class="font-medium text-gray-dark dark:text-white">${col.total_size}</span>
                </div>
                <div class="flex justify-between">
                    <span class="text-gray-light dark:text-gray-400">Indexed:</span>
                    <span class="font-medium text-gray-dark dark:text-white">${col.index_size}</span>
                </div>
                <div class="flex justify-between">
                    <span class="text-gray-light dark:text-gray-400">Memtable:</span>
                    <span class="font-medium text-gray-dark dark:text-white">${col.memtable_size}</span>
                </div>
            </div>
            <div class="flex gap-2">
                <button onclick="viewCollection('${name}')" class="flex-1 h-10 cursor-pointer rounded-lg bg-blue-strong text-white text-sm font-medium hover:bg-blue-strong/90">
                    View
                </button>
                <button onclick="commitCollection('${name}')" class="flex-1 h-10 cursor-pointer rounded-lg border border-gray-300 dark:border-gray-700 text-gray-dark dark:text-white text-sm font-medium hover:bg-gray-100 dark:hover:bg-gray-800">
                    Commit
                </button>
            </div>
            <button onclick="compactCollection('${name}')" class="w-full h-8 cursor-pointer rounded-lg border border-blue-strong text-blue-strong text-xs font-medium hover:bg-blue-light dark:hover:bg-blue-strong/20" title="Rewrite entire collection with current compression setting">
                <span class="material-symbols-outlined text-sm align-middle">compress</span>
                Compact
            </button>
        </div>
    `;
}

async function viewCollection(name) {
    currentCollection = name;
    try {
        const response = await fetch(`${API_BASE}/api/collections/${name}/all`);
        const toonData = await response.text();
        
        const countData = await fetchAPI(`/api/collections/${name}/count`);
        
        const documents = parseTOON(toonData);
        
        renderCollectionView(name, countData, documents);
    } catch (error) {
        showError('Failed to load collection: ' + error.message);
    }
}

function renderCollectionView(name, counts, documents) {
    const content = `
        <div class="flex items-center gap-4 mb-4">
            <button onclick="loadCollectionsPage()" class="flex h-10 w-10 items-center justify-center rounded-lg border border-gray-300 dark:border-gray-700 hover:bg-gray-100 dark:hover:bg-gray-800">
                <span class="material-symbols-outlined">arrow_back</span>
            </button>
            <div class="flex flex-col gap-1 flex-1">
                <p class="text-gray-dark dark:text-white text-3xl font-black leading-tight">${name}</p>
                <p class="text-gray-light dark:text-gray-400 text-base">Total: ${counts.total} | Indexed: ${counts.indexed} | Memtable: ${counts.memtable}</p>
            </div>
            <button onclick="showAddDocumentModal('${name}')" class="flex h-10 px-4 items-center justify-center gap-2 rounded-lg bg-green-status text-white text-sm font-medium hover:bg-green-status/90">
                <span class="material-symbols-outlined">add</span>
                Add Document
            </button>
            <button onclick="commitCollection('${name}')" class="flex h-10 px-4 items-center justify-center rounded-lg bg-blue-strong text-white text-sm font-medium hover:bg-blue-strong/90">
                Commit Changes
            </button>
            <button onclick="compactCollection('${name}')" class="flex h-10 px-4 items-center justify-center gap-2 rounded-lg border border-blue-strong text-blue-strong text-sm font-medium hover:bg-blue-light dark:hover:bg-blue-strong/20" title="Rewrite entire collection with current compression setting">
                <span class="material-symbols-outlined">compress</span>
                Compact
            </button>
        </div>
        
        <div class="rounded-xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-[#111a22] overflow-hidden">
            <div class="overflow-x-auto max-h-[600px]">
                <table class="w-full text-left text-sm">
                    <thead class="sticky top-0 bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800">
                        <tr>
                            ${documents.length > 0 ? Object.keys(documents[0]).map(key => 
                                `<th class="py-3 px-4 font-medium text-gray-light dark:text-gray-400">${key}</th>`
                            ).join('') : '<th class="py-3 px-4">No Data</th>'}
                            ${documents.length > 0 ? '<th class="py-3 px-4 font-medium text-gray-light dark:text-gray-400">Actions</th>' : ''}
                        </tr>
                    </thead>
                    <tbody class="text-gray-dark dark:text-gray-300">
                        ${documents.map((doc, idx) => `
                            <tr class="border-b border-gray-200 dark:border-gray-800 hover:bg-gray-50 dark:hover:bg-gray-900">
                                ${Object.values(doc).map(val => 
                                    `<td class="py-3 px-4">${formatValue(val)}</td>`
                                ).join('')}
                                <td class="py-3 px-4">
                                    <div class="flex gap-2">
                                        <button onclick='editDocument(${JSON.stringify(doc).replace(/'/g, "\\'")},"${name}")' class="text-blue-strong hover:underline" title="Edit">
                                            <span class="material-symbols-outlined text-lg">edit</span>
                                        </button>
                                        <button onclick='deleteDocument("${doc.id}", "${name}")' class="text-red-status hover:text-red-600" title="Delete">
                                            <span class="material-symbols-outlined text-lg">delete</span>
                                        </button>
                                    </div>
                                </td>
                            </tr>
                        `).join('')}
                    </tbody>
                </table>
            </div>
        </div>
        
        <!-- Modal Container -->
        <div id="modal-container"></div>
    `;
    
    document.getElementById('main-content').innerHTML = content;
}

function loadQueryPage() {
    currentPage = 'query';
    const content = `
        <div class="flex flex-col gap-1">
            <p class="text-gray-dark dark:text-white text-3xl font-black leading-tight">Query Editor</p>
            <p class="text-gray-light dark:text-gray-400 text-base font-normal leading-normal">Execute queries on your collections.</p>
        </div>
        
        <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
            <div class="flex flex-col gap-4 lg:col-span-2">
                <div class="rounded-xl border border-gray-200 dark:border-gray-800 p-6 bg-white dark:bg-[#111a22]">
                    <div class="space-y-4">
                        <div>
                            <label class="block text-sm font-medium text-gray-dark dark:text-white mb-2">Collection</label>
                            <select id="query-collection" class="w-full rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-4 py-2 text-gray-dark dark:text-white">
                                <option value="">Select a collection</option>
                                ${collections.map(name => `<option value="${name}">${name}</option>`).join('')}
                            </select>
                        </div>
                        <div>
                            <label class="block text-sm font-medium text-gray-dark dark:text-white mb-2">Field</label>
                            <input id="query-field" type="text" placeholder="e.g., age, price, name" class="w-full rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-4 py-2 text-gray-dark dark:text-white">
                        </div>
                        <div>
                            <label class="block text-sm font-medium text-gray-dark dark:text-white mb-2">Operator</label>
                            <select id="query-operator" class="w-full rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-4 py-2 text-gray-dark dark:text-white">
                                <option value="=">=</option>
                                <option value="!=">!=</option>
                                <option value=">">></option>
                                <option value="<"><</option>
                                <option value=">=">>=</option>
                                <option value="<="><=</option>
                            </select>
                        </div>
                        <div>
                            <label class="block text-sm font-medium text-gray-dark dark:text-white mb-2">Value</label>
                            <input id="query-value" type="text" placeholder="e.g., 30, Product 1" class="w-full rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-4 py-2 text-gray-dark dark:text-white">
                        </div>
                        <button onclick="executeQuery()" class="w-full h-10 rounded-lg bg-blue-strong text-white text-sm font-medium hover:bg-blue-strong/90">
                            Execute Query
                        </button>
                    </div>
                </div>
            </div>
            
            <div class="flex flex-col gap-4">
                <div class="rounded-xl border border-gray-200 dark:border-gray-800 p-6 bg-white dark:bg-[#111a22]">
                    <h3 class="text-gray-dark dark:text-white text-lg font-bold mb-4">Query Examples</h3>
                    <div class="space-y-2 text-sm">
                        <div class="p-2 bg-gray-50 dark:bg-gray-900 rounded">
                            <code class="text-blue-strong">price > 100</code>
                        </div>
                        <div class="p-2 bg-gray-50 dark:bg-gray-900 rounded">
                            <code class="text-blue-strong">name = Product 1</code>
                        </div>
                        <div class="p-2 bg-gray-50 dark:bg-gray-900 rounded">
                            <code class="text-blue-strong">in_stock = true</code>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        
        <div id="query-results" class="hidden rounded-xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-[#111a22] p-6">
            <h3 class="text-gray-dark dark:text-white text-lg font-bold mb-4">Results</h3>
            <div id="query-results-content"></div>
        </div>
    `;
    
    document.getElementById('main-content').innerHTML = content;
}

async function executeQuery() {
    const collection = document.getElementById('query-collection').value;
    const field = document.getElementById('query-field').value;
    const operator = document.getElementById('query-operator').value;
    const value = document.getElementById('query-value').value;
    
    if (!collection || !field || !value) {
        showError('Please fill in all fields');
        return;
    }
    
    try {
        const response = await fetch(`${API_BASE}/api/collections/${collection}/query`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ field, operator, value })
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Query failed');
        }
        
        const toonData = await response.text();
        const results = parseTOON(toonData);
        
        displayQueryResults(results, results.length);
    } catch (error) {
        showError('Query failed: ' + error.message);
    }
}

function displayQueryResults(results, count) {
    const resultsDiv = document.getElementById('query-results');
    const contentDiv = document.getElementById('query-results-content');
    
    resultsDiv.classList.remove('hidden');
    
    if (results.length === 0) {
        contentDiv.innerHTML = '<p class="text-gray-light dark:text-gray-400">No results found</p>';
        return;
    }
    
    contentDiv.innerHTML = `
        <p class="text-sm text-gray-light dark:text-gray-400 mb-4">Found ${count} result(s)</p>
        <div class="overflow-x-auto">
            <table class="w-full text-left text-sm">
                <thead class="border-b border-gray-200 dark:border-gray-800">
                    <tr>
                        ${Object.keys(results[0]).map(key => 
                            `<th class="py-2 px-3 font-medium text-gray-light dark:text-gray-400">${key}</th>`
                        ).join('')}
                    </tr>
                </thead>
                <tbody class="text-gray-dark dark:text-gray-300">
                    ${results.map(doc => `
                        <tr class="border-b border-gray-200 dark:border-gray-800">
                            ${Object.values(doc).map(val => 
                                `<td class="py-3 px-3">${formatValue(val)}</td>`
                            ).join('')}
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        </div>
    `;
}

function loadSettingsPage() {
    currentPage = 'settings';
    const content = `
        <div class="flex flex-col gap-1">
            <p class="text-gray-dark dark:text-white text-3xl font-black leading-tight">Settings</p>
            <p class="text-gray-light dark:text-gray-400 text-base font-normal leading-normal">Configure your FlyDB instance.</p>
        </div>
        
        <div class="rounded-xl border border-gray-200 dark:border-gray-800 p-6 bg-white dark:bg-[#111a22]">
            <h3 class="text-gray-dark dark:text-white text-lg font-bold mb-4">Database Configuration</h3>
            <div class="space-y-4">
                <div class="flex items-center justify-between py-3 border-b border-gray-200 dark:border-gray-800">
                    <div>
                        <p class="text-gray-dark dark:text-white font-medium">Data Directory</p>
                        <p class="text-sm text-gray-light dark:text-gray-400">${stats.data_dir || 'N/A'}</p>
                    </div>
                </div>
                <div class="flex items-center justify-between py-3 border-b border-gray-200 dark:border-gray-800">
                    <div>
                        <p class="text-gray-dark dark:text-white font-medium">Gzip Compression</p>
                        <p class="text-sm text-gray-light dark:text-gray-400">Compress new commits with gzip (applies to new data only)</p>
                    </div>
                    <label class="relative inline-flex items-center cursor-pointer">
                        <input type="checkbox" id="compression-toggle" ${compressionEnabled ? 'checked' : ''} onchange="toggleCompression()" class="sr-only peer">
                        <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 dark:peer-focus:ring-blue-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-blue-strong"></div>
                    </label>
                </div>
                <div class="flex items-center justify-between py-3">
                    <div>
                        <p class="text-gray-dark dark:text-white font-medium">API Endpoint</p>
                        <p class="text-sm text-gray-light dark:text-gray-400">${API_BASE}</p>
                    </div>
                </div>
            </div>
        </div>
        
        <div class="rounded-xl border border-gray-200 dark:border-gray-800 p-6 bg-white dark:bg-[#111a22]">
            <h3 class="text-gray-dark dark:text-white text-lg font-bold mb-4">Database Actions</h3>
            <div class="space-y-3">
                <button onclick="commitAllCollections()" class="w-full h-12 rounded-lg bg-blue-strong text-white text-sm font-medium hover:bg-blue-strong/90">
                    Commit All Collections
                </button>
                <button onclick="refreshData()" class="w-full h-12 rounded-lg border border-gray-300 dark:border-gray-700 text-gray-dark dark:text-white text-sm font-medium hover:bg-gray-100 dark:hover:bg-gray-800">
                    Refresh All Data
                </button>
            </div>
        </div>
    `;
    
    document.getElementById('main-content').innerHTML = content;
}


async function fetchAPI(endpoint, options = {}) {
    const response = await fetch(API_BASE + endpoint, {
        headers: {
            'Content-Type': 'application/json',
            ...options.headers
        },
        ...options
    });
    
    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Request failed');
    }
    
    return response.json();
}

async function commitCollection(name) {
    try {
        const data = await fetchAPI(`/api/collections/${name}/commit`, { method: 'POST' });
        showNotification(data.message);
        await refreshData();
    } catch (error) {
        showError('Commit failed: ' + error.message);
    }
}

async function commitAllCollections() {
    for (const name of collections) {
        try {
            await fetchAPI(`/api/collections/${name}/commit`, { method: 'POST' });
        } catch (error) {
            console.error(`Failed to commit ${name}:`, error);
        }
    }
    showNotification('All collections committed');
    await refreshData();
}

async function compactCollection(name) {
    if (!confirm(`Compact collection "${name}"? This will rewrite the entire collection file with the current compression setting.`)) {
        return;
    }
    
    try {
        const data = await fetchAPI(`/api/collections/${name}/compact`, { method: 'POST' });
        showNotification(data.message);
        await refreshData();
    } catch (error) {
        showError('Compact failed: ' + error.message);
    }
}

function formatValue(val) {
    if (val === null || val === undefined) return '<nil>';
    if (typeof val === 'boolean') return val ? 'true' : 'false';
    if (typeof val === 'object') return JSON.stringify(val);
    return String(val);
}

function showNotification(message) {
    console.log('Notification:', message);
    const notification = document.createElement('div');
    notification.className = 'fixed top-4 right-4 bg-green-status text-white px-6 py-3 rounded-lg shadow-lg z-50';
    notification.textContent = message;
    document.body.appendChild(notification);
    setTimeout(() => notification.remove(), 3000);
}

function showError(message) {
    console.error('Error:', message);
    const notification = document.createElement('div');
    notification.className = 'fixed top-4 right-4 bg-red-status text-white px-6 py-3 rounded-lg shadow-lg z-50';
    notification.textContent = message;
    document.body.appendChild(notification);
    setTimeout(() => notification.remove(), 5000);
}


function showAddDocumentModal(collection) {
    currentCollection = collection;
    const modal = `
        <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" onclick="closeModal(event)">
            <div class="bg-white dark:bg-[#111a22] rounded-xl p-6 max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto" onclick="event.stopPropagation()">
                <div class="flex items-center justify-between mb-4">
                    <h3 class="text-gray-dark dark:text-white text-xl font-bold">Add New Document</h3>
                    <button onclick="closeModal()" class="text-gray-light dark:text-gray-400 hover:text-gray-dark dark:hover:text-white">
                        <span class="material-symbols-outlined">close</span>
                    </button>
                </div>
                <div class="space-y-4">
                    <p class="text-sm text-gray-light dark:text-gray-400">Enter document data as JSON:</p>
                    <textarea id="document-json" rows="10" class="w-full rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-4 py-2 text-gray-dark dark:text-white font-mono text-sm" placeholder='{\n  "id": "unique_id",\n  "field1": "value1",\n  "field2": 123\n}'></textarea>
                    <div class="flex gap-3 justify-end">
                        <button onclick="closeModal()" class="px-4 py-2 rounded-lg border border-gray-300 dark:border-gray-700 text-gray-dark dark:text-white hover:bg-gray-100 dark:hover:bg-gray-800">
                            Cancel
                        </button>
                        <button onclick="saveNewDocument()" class="px-4 py-2 rounded-lg bg-blue-strong text-white hover:bg-blue-strong/90">
                            Add Document
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `;
    document.body.insertAdjacentHTML('beforeend', modal);
}

function editDocument(doc, collection) {
    currentCollection = collection;
    editingDocument = doc;
    const modal = `
        <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" onclick="closeModal(event)">
            <div class="bg-white dark:bg-[#111a22] rounded-xl p-6 max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto" onclick="event.stopPropagation()">
                <div class="flex items-center justify-between mb-4">
                    <h3 class="text-gray-dark dark:text-white text-xl font-bold">Edit Document</h3>
                    <button onclick="closeModal()" class="text-gray-light dark:text-gray-400 hover:text-gray-dark dark:hover:text-white">
                        <span class="material-symbols-outlined">close</span>
                    </button>
                </div>
                <div class="space-y-4">
                    <p class="text-sm text-gray-light dark:text-gray-400">Document ID: <strong>${doc.id}</strong></p>
                    <textarea id="document-json" rows="10" class="w-full rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-4 py-2 text-gray-dark dark:text-white font-mono text-sm">${JSON.stringify(doc, null, 2)}</textarea>
                    <div class="flex gap-3 justify-end">
                        <button onclick="closeModal()" class="px-4 py-2 rounded-lg border border-gray-300 dark:border-gray-700 text-gray-dark dark:text-white hover:bg-gray-100 dark:hover:bg-gray-800">
                            Cancel
                        </button>
                        <button onclick="saveEditedDocument()" class="px-4 py-2 rounded-lg bg-blue-strong text-white hover:bg-blue-strong/90">
                            Save Changes
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `;
    document.body.insertAdjacentHTML('beforeend', modal);
}

function closeModal(event) {
    if (!event || event.target === event.currentTarget) {
        const modals = document.querySelectorAll('.fixed.inset-0');
        modals.forEach(modal => modal.remove());
        editingDocument = null;
    }
}

async function saveNewDocument() {
    const jsonText = document.getElementById('document-json').value;
    
    try {
        const document = JSON.parse(jsonText);
        
        if (!document.id) {
            showError('Document must have an "id" field');
            return;
        }
        
        const data = await fetchAPI(`/api/collections/${currentCollection}/documents`, {
            method: 'POST',
            body: JSON.stringify({ document })
        });
        
        showNotification('Document added successfully');
        closeModal();
        viewCollection(currentCollection);
    } catch (error) {
        showError('Failed to add document: ' + error.message);
    }
}

async function saveEditedDocument() {
    const jsonText = document.getElementById('document-json').value;
    
    try {
        const document = JSON.parse(jsonText);
        const id = editingDocument.id;
        
        const data = await fetchAPI(`/api/collections/${currentCollection}/documents/${id}`, {
            method: 'PUT',
            body: JSON.stringify({ document })
        });
        
        showNotification('Document updated successfully');
        closeModal();
        viewCollection(currentCollection);
    } catch (error) {
        showError('Failed to update document: ' + error.message);
    }
}

async function toggleCompression() {
    const enabled = document.getElementById('compression-toggle').checked;
    
    try {
        await fetchAPI('/api/settings/compression', {
            method: 'POST',
            body: JSON.stringify({ enabled })
        });
        
        compressionEnabled = enabled;
        showNotification(`Compression ${enabled ? 'enabled' : 'disabled'}`);
    } catch (error) {
        showError('Failed to toggle compression: ' + error.message);
        document.getElementById('compression-toggle').checked = !enabled;
    }
}

function showCreateCollectionModal() {
    const modal = `
        <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" onclick="closeModal(event)">
            <div class="bg-white dark:bg-[#111a22] rounded-xl p-6 max-w-md w-full mx-4" onclick="event.stopPropagation()">
                <div class="flex items-center justify-between mb-4">
                    <h3 class="text-gray-dark dark:text-white text-xl font-bold">Create Collection</h3>
                    <button onclick="closeModal()" class="text-gray-light dark:text-gray-400 hover:text-gray-dark dark:hover:text-white">
                        <span class="material-symbols-outlined">close</span>
                    </button>
                </div>
                <div class="space-y-4">
                    <div>
                        <label class="block text-sm font-medium text-gray-dark dark:text-white mb-2">Collection Name</label>
                        <input id="collection-name" type="text" placeholder="e.g., users, products" class="w-full rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-4 py-2 text-gray-dark dark:text-white">
                    </div>
                    <div class="flex gap-3 justify-end">
                        <button onclick="closeModal()" class="px-4 py-2 rounded-lg border border-gray-300 dark:border-gray-700 text-gray-dark dark:text-white hover:bg-gray-100 dark:hover:bg-gray-800">
                            Cancel
                        </button>
                        <button onclick="createCollection()" class="px-4 py-2 rounded-lg bg-blue-strong text-white hover:bg-blue-strong/90">
                            Create
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `;
    document.body.insertAdjacentHTML('beforeend', modal);
    setTimeout(() => document.getElementById('collection-name').focus(), 100);
}

async function createCollection() {
    const name = document.getElementById('collection-name').value.trim();
    
    if (!name) {
        showError('Collection name is required');
        return;
    }
    
    if (!/^[a-zA-Z0-9_-]+$/.test(name)) {
        showError('Collection name can only contain letters, numbers, hyphens, and underscores');
        return;
    }
    
    try {
        await fetchAPI('/api/collections', {
            method: 'POST',
            body: JSON.stringify({ name })
        });
        
        showNotification(`Collection "${name}" created successfully`);
        closeModal();
        loadCollectionsPage();
        loadDashboard();
    } catch (error) {
        showError('Failed to create collection: ' + error.message);
    }
}

async function deleteCollection(name) {
    if (!confirm(`Are you sure you want to delete the collection "${name}"? This action cannot be undone.`)) {
        return;
    }
    
    try {
        await fetchAPI(`/api/collections/${name}`, {
            method: 'DELETE'
        });
        
        showNotification(`Collection "${name}" deleted successfully`);
        loadCollectionsPage();
        loadDashboard();
    } catch (error) {
        showError('Failed to delete collection: ' + error.message);
    }
}

async function deleteDocument(id, collection) {
    if (!confirm(`Are you sure you want to delete document "${id}"?`)) {
        return;
    }
    
    try {
        await fetchAPI(`/api/collections/${collection}/documents/${id}`, {
            method: 'DELETE'
        });
        
        showNotification(`Document "${id}" deleted successfully`);
        viewCollection(collection);
    } catch (error) {
        showError('Failed to delete document: ' + error.message);
    }
}

function parseTOON(toonText) {
    const lines = toonText.trim().split('\n');
    
    if (lines.length === 0) return [];
    
    const headerLine = lines[0];
    const headerMatch = headerLine.match(/^(\w+)\[(\d+)\]\{([^}]*)\}:$/);
    
    if (!headerMatch) {
        console.error('Invalid TOON header:', headerLine);
        return [];
    }
    
    const [, collectionName, count, schemaStr] = headerMatch;
    const numDocs = parseInt(count, 10);
    
    if (numDocs === 0) return [];
    
    const schema = schemaStr.split(',').map(s => s.trim()).filter(s => s.length > 0);
    
    if (schema.length === 0) {
        console.error('Empty schema in TOON header');
        return [];
    }
    
    const documents = [];
    
    for (let i = 1; i <= numDocs && i < lines.length; i++) {
        const line = lines[i];
        if (!line.trim()) continue;
        
        const values = parseCSVLine(line);
        
        const doc = {};
        schema.forEach((key, idx) => {
            if (idx < values.length) {
                doc[key] = unescapeTOON(values[idx]);
            }
        });
        
        documents.push(doc);
    }
    
    return documents;
}

function parseCSVLine(line) {
    const values = [];
    let current = '';
    let escaped = false;
    
    for (let i = 0; i < line.length; i++) {
        const char = line[i];
        
        if (escaped) {
            current += char;
            escaped = false;
        } else if (char === '\\') {
            escaped = true;
        } else if (char === ',') {
            values.push(current);
            current = '';
        } else {
            current += char;
        }
    }
    
    values.push(current);
    return values;
}

function unescapeTOON(s) {
    return s
        .replace(/\\n/g, '\n')
        .replace(/\\r/g, '\r')
        .replace(/\\,/g, ',')
        .replace(/\\\\/g, '\\');
}
