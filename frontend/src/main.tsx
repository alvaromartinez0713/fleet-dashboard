import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import config from '@config/environment';

// Locate the root DOM element where React will mount the application
const rootElement = document.getElementById('root');

if (!rootElement) {
  throw new Error(
    'Failed to find root DOM element. Ensure index.html contains <div id="root"></div>',
  );
}

// Mount the React application using React 18+ API
ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);

console.log('✅ Fleet Dashboard loaded');
console.log('📊 Data Mode:', config.dataMode);
console.log('🗺️ Mapbox Token:', config.mapboxToken ? 'Configured' : 'Not configured');

