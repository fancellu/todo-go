// Theme toggling functionality.
// The saved theme is applied by an inline script in <head> (before first paint),
// which adds the "dark" class to <html>. This file syncs the icon and handles clicks.
document.addEventListener('DOMContentLoaded', function() {
    const themeToggle = document.getElementById('theme-toggle');
    const themeIcon = document.getElementById('theme-icon');

    if (!themeToggle || !themeIcon) return;

    const root = document.documentElement;

    function applyTheme(isDark) {
        root.classList.toggle('dark', isDark);
        themeIcon.classList.toggle('bi-moon-fill', isDark);
        themeIcon.classList.toggle('bi-brightness-high-fill', !isDark);
    }

    applyTheme(root.classList.contains('dark'));

    themeToggle.addEventListener('click', () => {
        const isDark = !root.classList.contains('dark');
        applyTheme(isDark);
        try {
            localStorage.setItem('theme', isDark ? 'dark' : 'light');
        } catch (e) {}
    });
});
