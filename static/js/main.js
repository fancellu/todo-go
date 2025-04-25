// Main JavaScript file
document.addEventListener('DOMContentLoaded', function() {
    // Task form submission handling
    const taskForm = document.getElementById('task-form');
    if (taskForm) {
        taskForm.addEventListener('submit', function(event) {
            const titleInput = this.querySelector('input[name="title"]');
            if (!titleInput.value.trim()) {
                event.preventDefault();
                alert('Task title cannot be empty');
            }
        });
    }

    // Task checkbox handling in task.html
    const completedCheckbox = document.getElementById('completedCheckbox');
    if (completedCheckbox) {
        completedCheckbox.addEventListener('change', function() {
            this.closest('form').submit();
        });
    }
});