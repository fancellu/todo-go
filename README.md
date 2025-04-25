# Simple Todo List

This is a simple web-based Golang Todo List application using GIN that allows you to manage your tasks efficiently.

## Features

*   **Add Tasks:** Easily add new tasks to your todo list.
*   **Mark as Complete:** Mark tasks as complete, which will display them with a strikethrough.
*   **Delete Tasks:** Remove tasks from your list when they are no longer needed.
*   **Edit Tasks:** Modify the title of existing tasks.
*   **Dark/Light Theme:** Switch between a light and a dark theme for comfortable viewing.
*   **Validation:** The app has validation to prevent adding empty tasks.
*   **Press Enter to Commit:** In the edit task input field, you can press Enter to commit.

## How to Run

1.  **Install Go:** If you don't have Go installed, download and install it from the official Go website: [https://go.dev/](https://go.dev/)
2.  **Run the App:** Open your terminal and navigate to the project directory. Then, run the following command:
```
    go run main.go
```
3.  **Open in Browser:** Open your web browser and go to `http://localhost:8080` to access the application.

## How to Use

*   **Add a Task:** Enter the task title in the input field and click "Add Task". The app has validation to prevent adding empty tasks.
*   **Mark as Complete:** Click the check icon next to a task to mark it as complete. Click again to uncomplete it.
*   **Delete a Task:** Click the trashcan icon to delete a task.
*   **Edit a Task:**
    *   Click the pencil icon next to the task you want to edit.
    *   The task title will be replaced with an input field.
    *   Modify the text, and press Enter or click the save icon (a carriage return arrow) to save your changes.
*   **Toggle Theme:** Click the sun/moon icon in the top right corner to switch between light and dark themes.