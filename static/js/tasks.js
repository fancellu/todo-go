// Task-related functionality
document.addEventListener('DOMContentLoaded', function() {
    // Edit button functionality
    document.querySelectorAll('.edit-button').forEach(button => {
        button.addEventListener('click', () => {
            const listItem = button.closest('.list-group-item');
            const span = listItem.querySelector('span');
            const input = listItem.querySelector('.edit-input');
            const saveButton = listItem.querySelector('.save-button');
            span.style.display = 'none';
            input.style.display = 'inline-block';
            saveButton.style.display = 'inline-block';
        });
    });

    // Edit input functionality
    document.querySelectorAll('.edit-input').forEach(input => {
        input.addEventListener('keydown', function(event) {
            if (event.key === 'Enter') {
                const taskId = this.getAttribute('data-task-id');
                updateTaskTitle(taskId, this.value);
            }
        });
    });

    // Save button functionality
    document.querySelectorAll('.save-button').forEach(button => {
        button.addEventListener('click', function() {
            const listItem = this.closest('.list-group-item');
            const input = listItem.querySelector('.edit-input');
            const taskId = input.getAttribute('data-task-id');
            updateTaskTitle(taskId, input.value);
        });
    });

    // Toggle completion functionality
    document.querySelectorAll('.toggle-complete-button').forEach(button => {
        button.addEventListener('click', function() {
            const taskId = this.getAttribute('data-task-id');
            const completed = this.getAttribute('data-completed') === 'true' ? 'false' : 'true';
            toggleTaskCompletion(taskId, completed);
        });
    });

    // Delete task functionality
    document.querySelectorAll('.delete-task-button').forEach(button => {
        button.addEventListener('click', function() {
            const taskId = this.getAttribute('data-task-id');
            deleteTask(taskId);
        });
    });
});

// Function to update task title
function updateTaskTitle(taskId, title) {
    fetch(`/task/${taskId}/title`, { 
        method: 'PUT', 
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' }, 
        body: `title=${encodeURIComponent(title)}`
    })
    .then(response => { 
        if (response.ok) { 
            window.location.reload(); 
        } 
    })
    .catch(error => console.error('Error:', error));
}

// Function to toggle task completion
function toggleTaskCompletion(taskId, completed) {
    fetch(`/task/${taskId}`, { 
        method: 'PUT', 
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' }, 
        body: `completed=${completed}`
    })
    .then(response => { 
        if (response.ok) { 
            window.location.reload(); 
        } 
    })
    .catch(error => console.error('Error:', error));
}

// Function to delete task
function deleteTask(taskId) {
    fetch(`/task/${taskId}`, { 
        method: 'DELETE' 
    })
    .then(response => {
        if (response.ok) { 
            window.location.reload(); 
        } 
    })
    .catch(error => console.error('Error:', error));
}