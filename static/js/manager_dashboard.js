function toggleWorksiteField() {
    var role = document.getElementById("role").value;
    var worksiteDiv = document.getElementById("worksiteDiv");
    console.log(role)
    if (role === "worker") {
        worksiteDiv.classList.remove("hidden");
    } else {
        worksiteDiv.classList.add("hidden");
    }
}

function toggleEditWorksiteField() {
    const role = document.getElementById('editrole').value;
    const worksiteDiv = document.getElementById('editWorksiteDiv');
    const worksiteSelect = document.getElementById('editWorksite');

    if (role === 'worker') {
        worksiteDiv.classList.remove('visible');
        worksiteSelect.required = true;
    } else {
        worksiteDiv.classList.add('visible');
        worksiteSelect.required = false;
    }
}


function initializeTable() {
    console.log('Initializing table...');

    const table = document.getElementById('usersTable');
    if (!table) {
        console.error('Table not found');
        return;
    }

    if (!document.querySelector('.filter-container')) {
        const filterContainer = document.createElement('div');
        filterContainer.classList.add('filter-container');
        filterContainer.innerHTML = `
            <div class="search-container">
                <input type="text" id="searchInput" placeholder="Vyhľadávanie...">
            </div>
            <div class="filter-controls">
                <select id="roleFilter">
                    <option value="">Všetky pozície</option>
                    <option value="salesman">Obchoďák</option>
                    <option value="admin">AD pracovník</option>
                    <option value="worker">Výroba</option>
                </select>
                <select id="worksiteFilter">
                    <option value="">Všetky pracoviská</option>
                    <option value="sypke">Sypké</option>
                    <option value="pozivatiny">Poživatiny</option>
                    <option value="kozmetika">Kozmetika</option>
                    <option value="sklad">Sklad</option>
                </select>
            </div>
        `;
        table.parentNode.insertBefore(filterContainer, table);
    }

    const tbody = table.querySelector('tbody');
    if (!tbody) {
        console.error('Table body not found');
        return;
    }

    let sortState = {
        column: null,
        asc: true
    };

    const headers = table.querySelectorAll('thead th');
    headers.forEach((header, index) => {
        if (header.classList.contains('no-sort')) return;

        const newHeader = header.cloneNode(true);
        header.parentNode.replaceChild(newHeader, header);

        newHeader.addEventListener('click', () => {
            console.log('Header clicked:', index);
            sortTable(index);
        });
    });

    function sortTable(columnIndex) {
        const rows = Array.from(tbody.querySelectorAll('tr'));

        if (sortState.column === columnIndex) {
            sortState.asc = !sortState.asc;
        } else {
            sortState.column = columnIndex;
            sortState.asc = true;
        }

        headers.forEach(th => th.classList.remove('sort-asc', 'sort-desc'));
        headers[columnIndex].classList.add(sortState.asc ? 'sort-asc' : 'sort-desc');

        rows.sort((rowA, rowB) => {
            const cellA = rowA.cells[columnIndex].textContent.trim();
            const cellB = rowB.cells[columnIndex].textContent.trim();

            if (!isNaN(cellA) && !isNaN(cellB)) {
                return sortState.asc ?
                    Number(cellA) - Number(cellB) :
                    Number(cellB) - Number(cellA);
            }

            return sortState.asc ?
                cellA.localeCompare(cellB) :
                cellB.localeCompare(cellA);
        });

        rows.forEach(row => tbody.appendChild(row));
    }

    function filterTable() {
        const searchValue = document.getElementById('searchInput')?.value.toLowerCase() || '';
        const roleValue = document.getElementById('roleFilter')?.value.toLowerCase() || '';
        const worksiteValue = document.getElementById('worksiteFilter')?.value.toLowerCase() || '';

        const rows = tbody.querySelectorAll('tr');

        rows.forEach(row => {
            const cells = Array.from(row.cells);
            const rowText = cells.map(cell => cell.textContent.toLowerCase());

            const matchesSearch = searchValue === '' ||
                rowText.some(text => text.includes(searchValue));

            const matchesRole = roleValue === '' ||
                (cells[4] && cells[4].textContent.toLowerCase().includes(roleValue));

            const matchesWorksite = worksiteValue === '' ||
                (cells[6] && cells[6].textContent.toLowerCase().includes(worksiteValue));

            row.style.display = matchesSearch && matchesRole && matchesWorksite ? '' : 'none';
        });
    }

    document.getElementById('searchInput')?.addEventListener('input', filterTable);
    document.getElementById('roleFilter')?.addEventListener('change', filterTable);
    document.getElementById('worksiteFilter')?.addEventListener('change', filterTable);
}

function deleteProduct(id) {
    fetch('/delete-product', {
        method: 'POST',
        headers: {
            'Content-Type': 'text/plain'
        },
        body: id.toString()
    })
}
function deleteUser(id) {
    fetch('/delete-user', {
        method: 'POST',
        headers: {
            'Content-Type': 'text/plain'
        },
        body: id.toString()
    })
        .then(response => {
            if (response.ok) {
                const row = document.querySelector(`#user-${id}`);
                if (row) {
                    row.remove();
                } else {
                    console.error(`Row with ID user-${id} not found.`);
                }
            } else {
                console.error('Failed to delete user. Server responded with:', response.statusText);
            }
        })
        .catch(error => {
            console.error('An error occurred while deleting the user:', error);
        });
}



document.addEventListener('DOMContentLoaded', initializeTable);

document.body.addEventListener('htmx:afterOnLoad', function () {
    console.log('HTMX content loaded');
    initializeTable();
});

document.body.addEventListener('htmx:afterSwap', function () {
    console.log('HTMX content swapped');
    initializeTable();
});




htmx.on("htmx:configRequest", function (evt) {
    console.log("HTMX request:", evt.detail);
});
htmx.on("htmx:responseError", function (evt) {
    console.error("HTMX error:", evt.detail);
});

function enableRowEdit(row, event) {
    if (event) {
        event.preventDefault();
        event.stopPropagation();
    }

    const editableCells = row.querySelectorAll('.editable-cell');
    const userId = row.id.replace('user-', '');

    const actionCell = row.querySelector('td:last-child');
    const originalButtons = actionCell.innerHTML;
    actionCell.innerHTML = `
        <button onclick="saveRowEdit(this.closest('tr'), event)" class="save-btn">Uložiť</button>
        <button onclick="cancelRowEdit(this.closest('tr'), event)" class="cancel-btn">Zrušiť</button>
    `;

    let worksiteCell;
    editableCells.forEach(cell => {
        const originalValue = cell.textContent.trim();
        cell.dataset.originalValue = originalValue;

        if (cell.dataset.field === 'role') {
            const select = document.createElement('select');
            select.className = 'inline-edit-input';
            select.name = `role-${userId}`;

            const options = [
                { value: 'salesman', text: 'salesman' },
                { value: 'admin', text: 'admin' },
                { value: 'worker', text: 'worker' }
            ];
            options.forEach(opt => {
                const option = document.createElement('option');
                option.value = opt.value;
                option.text = opt.text;
                option.selected = opt.text === originalValue;
                select.appendChild(option);
            });

            select.addEventListener('change', function () {
                toggleWorksiteField(row, select.value);
            });

            cell.innerHTML = '';
            cell.appendChild(select);
        }

        else if (cell.dataset.field === 'worksite') {
            worksiteCell = cell;
            const select = document.createElement('select');
            select.className = 'inline-edit-input';
            const options = [
                { value: 'sypke', text: 'Sypké' },
                { value: 'pozivatiny', text: 'Poživatiny' },
                { value: 'kozmetika', text: 'Kozmetika' },
                { value: 'sklad', text: 'Sklad' }
            ];
            options.forEach(opt => {
                const option = document.createElement('option');
                option.value = opt.value;
                option.text = opt.text;
                option.selected = opt.text === originalValue;
                select.appendChild(option);
            });
            select.addEventListener('click', function (e) {
                e.stopPropagation();
            });
            cell.innerHTML = '';
            cell.appendChild(select);
        }

        else {
            const input = document.createElement('input');
            input.value = originalValue;
            input.className = 'inline-edit-input';

            input.addEventListener('click', function (e) {
                e.stopPropagation();
            });
            cell.innerHTML = '';
            cell.appendChild(input);
        }

        cell.addEventListener('click', function (e) {
            e.stopPropagation();
        });
    });

    const roleCell = row.querySelector('[data-field="role"] select');
    if (roleCell) {
        toggleWorksiteField(row, roleCell.value);
    }

    row.dataset.originalButtons = originalButtons;

    row.addEventListener('click', function (e) {
        e.stopPropagation();
    });

    function toggleWorksiteField(row, roleValue) {
        if (!worksiteCell) return;
        const select = worksiteCell.querySelector('select');
        if (roleValue === 'worker') {
            worksiteCell.style.visibility = 'visible'; // Make it visible
            worksiteCell.style.width = ''; // Restore width
            if (select) select.disabled = false; // Enable selection
        } else {
            worksiteCell.style.visibility = 'hidden'; // Hide it but retain structure
            worksiteCell.style.width = '0'; // Shrink width
            if (select) {
                select.disabled = true; // Disable selection
                select.value = ''; // Reset value
            }
        }
    }
}
function saveRowEdit(row, event) {
    if (event) {
        event.preventDefault();
        event.stopPropagation();
    }

    const userId = row.id.replace('user-', '');
    const formData = new FormData();
    formData.append('userId', userId);

    let isValid = true;
    const roleValue = row.querySelector('[data-field="role"] select')?.value;
    const worksiteCell = row.querySelector('[data-field="worksite"]');
    const worksiteSelect = worksiteCell?.querySelector('select');
    const worksiteValue = worksiteSelect ? worksiteSelect.value : null;

    if (roleValue === 'worker' && !worksiteValue) {
        isValid = false;
        alert('Please select a worksite for workers.');
        worksiteSelect?.focus(); // Highlight the worksite field
        return; // Stop the save operation
    }

    row.querySelectorAll('.editable-cell').forEach(cell => {
        const input = cell.querySelector('input, select');
        if (input) {
            if (input.tagName === 'SELECT') {
                formData.append(cell.dataset.field, input.value);
            } else {
                formData.append(cell.dataset.field, input.value);
            }
        }
    });

    fetch('/edit-user', {
        method: 'PUT',
        body: formData
    })
        .then(response => {
            if (!response.ok) throw new Error('Update failed');
            return response.text();
        })
        .then(() => {
            row.querySelectorAll('.editable-cell').forEach(cell => {
                const input = cell.querySelector('input, select');
                if (input) {
                    if (cell.dataset.field === 'password') {
                        // Keep the password field empty
                        cell.textContent = '';
                    } else if (input.tagName === 'SELECT') {
                        const select = cell.querySelector('select');
                        if (select && select.options[select.selectedIndex]) {
                            cell.textContent = select.options[select.selectedIndex].text;
                        }
                    } else {
                        cell.textContent = input.value;
                    }
                }
            });

            const actionCell = row.querySelector('td:last-child');
            actionCell.innerHTML = row.dataset.originalButtons;
        })
        .catch(error => {
            console.error('Error:', error);
            alert('Failed to update. Please try again.');
            cancelRowEdit(row, event);
        });
}

function cancelRowEdit(row, event) {
    if (event) {
        event.preventDefault();
        event.stopPropagation();
    }

    row.querySelectorAll('.editable-cell').forEach(cell => {
        cell.textContent = cell.dataset.originalValue;
    });

    const actionCell = row.querySelector('td:last-child');
    actionCell.innerHTML = row.dataset.originalButtons;
}


function enableProductRowEdit(row, event) {
    // Prevent event from bubbling up
    if (event) {
        event.preventDefault();
        event.stopPropagation();
    }

    const editableCells = row.querySelectorAll('.editable-cell');
    const productId = row.id.replace('product-', '');

    // Create save and cancel buttons
    const actionCell = row.querySelector('td:last-child');
    const originalButtons = actionCell.innerHTML;
    actionCell.innerHTML = `
        <button onclick="saveProductRowEdit(this.closest('tr'), event)" class="save-btn">Uložiť</button>
        <button onclick="cancelProductRowEdit(this.closest('tr'), event)" class="cancel-btn">Zrušiť</button>
    `;

    // Store original values and make cells editable
    editableCells.forEach(cell => {
        const originalValue = cell.textContent.trim();
        cell.dataset.originalValue = originalValue;

        if (cell.dataset.field === 'weight') {
            // Parse the weight value and type from the original content
            const weightValue = cell.querySelector('.weight-value').textContent.trim();
            const weightType = cell.querySelector('.weight-type').textContent.trim();

            cell.innerHTML = `
                <div class="weight-edit-container">
                    <input type="number" class="inline-edit-input weight-value-input" value="${weightValue}" min="0" step="0.01">
                    <select class="inline-edit-input weight-type-input">
                        <optgroup label="Hmotnosť">
                            <option value="g" ${weightType === 'g' ? 'selected' : ''}>g</option>
                            <option value="kg" ${weightType === 'kg' ? 'selected' : ''}>kg</option>
                        </optgroup>
                        <optgroup label="Objem">
                            <option value="ml" ${weightType === 'ml' ? 'selected' : ''}>ml</option>
                            <option value="l" ${weightType === 'l' ? 'selected' : ''}>l</option>
                        </optgroup>
                    </select>
                </div>
            `;
        } else {
            const input = document.createElement('input');
            input.value = originalValue;
            input.className = 'inline-edit-input';

            // Add event listeners to prevent event bubbling
            input.addEventListener('click', function (e) {
                e.stopPropagation();
            });
            cell.innerHTML = '';
            cell.appendChild(input);
        }
        // Add click event listener to the cell itself
        cell.addEventListener('click', function (e) {
            e.stopPropagation();
        });
    });

    // Store original buttons for restoration
    row.dataset.originalButtons = originalButtons;
}
function saveProductRowEdit(row, event) {
    if (event) {
        event.preventDefault();
        event.stopPropagation();
    }

    const productId = row.id.replace('product-', '');
    const formData = new FormData();
    formData.append('productId', productId);

    // Collect values from all inputs
    row.querySelectorAll('.editable-cell').forEach(cell => {
        if (cell.dataset.field === 'weight') {
            const weightValue = cell.querySelector('.weight-value-input').value;
            const weightType = cell.querySelector('.weight-type-input').value;
            formData.append('weight', weightValue);
            formData.append('weightType', weightType);
        } else {
            const input = cell.querySelector('input');
            if (input) {
                formData.append(cell.dataset.field, input.value);
            }
        }
    });
    console.log(weightType)
    console.log(weightValue)
    // Send update to server
    fetch('/edit-product', {
        method: 'PUT',
        body: formData
    })
        .then(response => {
            if (response.ok) {
                return response.text();
            }
            throw new Error('Update failed');
        })
        .then(() => {
            // Update the display values
            row.querySelectorAll('.editable-cell').forEach(cell => {
                if (cell.dataset.field === 'weight') {
                    const weightValue = cell.querySelector('.weight-value-input').value;
                    const weightType = cell.querySelector('.weight-type-input').value;
                    cell.innerHTML = `
                    <span class="weight-value">${weightValue}</span>
                    <span class="weight-type">${weightType}</span>
                `;
                } else {
                    const input = cell.querySelector('input');
                    if (input) {
                        cell.textContent = input.value;
                    }
                }
            });

            // Restore original buttons
            const actionCell = row.querySelector('td:last-child');
            actionCell.innerHTML = row.dataset.originalButtons;
        })
        .catch(error => {
            console.error('Error:', error);
            alert('Failed to update. Please try again.');
            cancelProductRowEdit(row, event);
        });
}
function cancelProductRowEdit(row, event) {
    if (event) {
        event.preventDefault();
        event.stopPropagation();
    }

    // Restore original values
    row.querySelectorAll('.editable-cell').forEach(cell => {
        cell.textContent = cell.dataset.originalValue;
    });

    // Restore original buttons
    const actionCell = row.querySelector('td:last-child');
    actionCell.innerHTML = row.dataset.originalButtons;
}


// Add additional CSS styles for product editing
const productStyle = document.createElement('style');
productStyle.textContent = `
    .weight-edit-container {
        display: flex;
        gap: 8px;
        align-items: center;
    }
    
    .weight-value-input {
        width: 80px;
    }
    
    .weight-type-input {
        width: 70px;
    }
`;
document.head.appendChild(productStyle);


// Add CSS styles for inline editing
const style = document.createElement('style');
style.textContent = `
    .inline-edit-input {
        width: 90%;
        padding: 4px;
        border: 1px solid #007bff;
        border-radius: 4px;
        box-sizing: border-box;
    }
    
    .editable-cell {
        padding: 4px;
    }
    
    .save-btn {
        background-color: #28a745;
        color: white;
        border: none;
        padding: 4px 8px;
        border-radius: 4px;
        cursor: pointer;
        margin-right: 4px;
    }
    
    .cancel-btn {
        background-color: #dc3545;
        color: white;
        border: none;
        padding: 4px 8px;
        border-radius: 4px;
        cursor: pointer;
    }
    
    .save-btn:hover {
        background-color: #218838;
    }
    
    .cancel-btn:hover {
        background-color: #c82333;
    }
`;
document.head.appendChild(style);




const editStyle = document.createElement('style');
editStyle.textContent = `
    .inline-edit-input {
        width: 100%;
        padding: 4px;
        border: 1px solid #007bff;
        border-radius: 4px;
        box-sizing: border-box;
        background-color: white;
    }
    
    .editable-cell {
        padding: 4px;
        role: relative;
    }
    
    .editable-cell input,
    .editable-cell select {
        z-index: 1;
    }
    
    .save-btn,
    .cancel-btn {
        z-index: 2;
    }
`;
document.head.appendChild(editStyle);
