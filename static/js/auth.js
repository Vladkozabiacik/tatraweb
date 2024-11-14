async function checkIfLoggedIn() {
    try {
        const response = await fetch('/status', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            credentials: 'same-origin',
            body: JSON.stringify({})
        });

        if (!response.ok) {
            console.log("Not logged in or error occurred");
        } else {
            const result = await response.json();
            if (result.loggedIn) {
                const userRole = result.role;
                window.location.href = `/${userRole}`;
            } else {
                console.log("User is not logged in");
            }
        }
    } catch (error) {
        console.error("Error checking login status:", error);
    }
}

window.onload = checkIfLoggedIn;

const urlParams = new URLSearchParams(window.location.search);
const error = urlParams.get('error');

if (error) {
    const errorMessageElement = document.getElementById('error-message');
    errorMessageElement.style.display = 'block';
    errorMessageElement.textContent = decodeURIComponent(error);
}