// Theme toggling functionality
document.addEventListener('DOMContentLoaded', function() {
    const themeToggle = document.getElementById('theme-toggle');
    const themeIcon = document.getElementById('theme-icon');
    
    if (!themeToggle || !themeIcon) return;
    
    let isDark = localStorage.getItem('theme') === 'dark';

    // Set the theme at load
    if (isDark) {
        document.body.classList.add('dark');
        themeIcon.classList.remove('bi-brightness-high-fill');
        themeIcon.classList.add('bi-moon-fill');
    } else {
        document.body.classList.remove('dark');
        themeIcon.classList.remove('bi-moon-fill');
        themeIcon.classList.add('bi-brightness-high-fill');
    }

    themeToggle.addEventListener('click', () => {
        if (isDark) {
            document.body.classList.remove('dark');
            themeIcon.classList.remove('bi-moon-fill');
            themeIcon.classList.add('bi-brightness-high-fill');
            localStorage.setItem('theme', 'light');
        } else {
            document.body.classList.add('dark');
            themeIcon.classList.remove('bi-brightness-high-fill');
            themeIcon.classList.add('bi-moon-fill');
            localStorage.setItem('theme', 'dark');
        }
        isDark = !isDark;
    });
});