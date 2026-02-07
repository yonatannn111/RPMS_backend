const https = require('https');

const url = 'https://rpms-backend.onrender.com/health'; // Using a health endpoint if available, or just root
const interval = 14 * 60 * 1000; // 14 minutes (Render sleeps after 15 mins of inactivity)

function ping() {
    console.log(`[${new Date().toISOString()}] Pinging ${url}...`);
    https.get(url, (res) => {
        console.log(`Status: ${res.statusCode}`);
    }).on('error', (e) => {
        console.error(`Error: ${e.message}`);
    });
}

ping(); // Initial ping
setInterval(ping, interval);
