function connectSSE() {
    const eventSource = new EventSource('${REFRESH_LIVE_RELOAD_SSE_URL}');

    eventSource.addEventListener('refresh-restart', function (event) {
        window.location.reload();
    });

    // Handle any errors that occur
    eventSource.onerror = function (error) {
        console.warn('refresh: EventSource failed:', error);
        eventSource.close(); // Close the current connection

        // Attempt to reconnect after a short delay
        setTimeout(function () {
            console.debug('refresh: Attempting to reconnect...');
            connectSSE(); // Attempt to reconnect
        }, 5000); // Reconnect after 5 seconds
    };

    // Clean up the event source when the page is closed or reloaded
    window.onbeforeunload = () => {
        eventSource.close();
    };
}

connectSSE();
