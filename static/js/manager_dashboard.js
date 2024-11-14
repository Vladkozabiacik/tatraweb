function toggleWorksiteField() {
    var position = document.getElementById("position").value;
    var worksiteDiv = document.getElementById("worksiteDiv");
    console.log(position)
    if (position === "worker") {
        worksiteDiv.classList.remove("hidden");
    } else {
        worksiteDiv.classList.add("hidden");
    }
}


// Update the JavaScript functions to use the correct endpoints
function showEditModal(userId, userLogin) {
    const modal = document.getElementById('editModal');
    modal.classList.add('visible');
    
    // Fetch user details using the correct endpoint
    fetch(`/get-user?id=${userId}`)
        .then(response => response.json())
        .then(user => {
            document.getElementById('editUserId').value = user.id;
            document.getElementById('editFirstName').value = user.firstName;
            document.getElementById('editLastName').value = user.lastName;
            document.getElementById('editLogin').value = user.login;
            document.getElementById('editPosition').value = user.role;
            
            if (user.role === 'worker') {
                document.getElementById('editWorksiteDiv').classList.remove('hidden');
                document.getElementById('editWorksite').value = user.worksite;
            } else {
                document.getElementById('editWorksiteDiv').classList.add('hidden');
            }
        })
        .catch(error => {
            console.error('Error fetching user details:', error);
            alert('Error fetching user details. Please try again.');
            modal.classList.remove('visible');
        });
}

function showDeleteModal(userId, userLogin) {
    const modal = document.getElementById('deleteModal');
    modal.classList.add('visible');
    
    document.getElementById('deleteUserId').value = userId;
    document.getElementById('expectedLogin').value = userLogin;
    
    const confirmInput = document.getElementById('confirmLogin');
    const deleteButton = document.querySelector('#deleteUserForm button');
    
    confirmInput.value = '';
    deleteButton.disabled = true;
    
    confirmInput.addEventListener('input', function() {
        deleteButton.disabled = this.value !== userLogin;
    });
}

function toggleEditWorksiteField() {
    const position = document.getElementById('editPosition').value;
    const worksiteDiv = document.getElementById('editWorksiteDiv');
    const worksiteSelect = document.getElementById('editWorksite');
    
    if (position === 'worker') {
        worksiteDiv.classList.remove('visible');
        worksiteSelect.required = true;
    } else {
        worksiteDiv.classList.add('visible');
        worksiteSelect.required = false;
    }
}

// Close modal when clicking the close button or outside the modal
document.querySelectorAll('.modal .close-button').forEach(button => {
    button.addEventListener('click', function() {
        this.closest('.modal').classList.remove('visible');
    });
});

document.querySelectorAll('.modal').forEach(modal => {
    modal.addEventListener('click', function(e) {
        if (e.target === this) {
            this.classList.remove('visible');
        }
    });
});

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

document.addEventListener('DOMContentLoaded', initializeTable);

document.body.addEventListener('htmx:afterOnLoad', function() {
    console.log('HTMX content loaded');
    initializeTable();
});

document.body.addEventListener('htmx:afterSwap', function() {
    console.log('HTMX content swapped');
    initializeTable();
});

function showPanel(panelId) {
    const panels = document.querySelectorAll('.panel');
    panels.forEach(panel => {
        panel.classList.remove('visible');
    });
    document.getElementById(panelId).classList.add('visible');
}

function toggleModal(modalId) {
    const modal = document.getElementById(modalId);
    modal.classList.toggle('visible');
}
