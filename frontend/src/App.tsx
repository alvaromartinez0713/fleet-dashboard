import React from 'react';
import { ApplicationProvider } from '@presentation/context/ApplicationProvider';
import { VehicleDashboard } from '@presentation/pages/VehicleDashboard';

/**
 * App - Root component of the Fleet Dashboard application
 *
 * Responsibilities:
 * - Wraps application with ApplicationProvider for dependency injection
 * - Renders VehicleDashboard as main content
 * - Provides global error boundary (future enhancement)
 * - Manages top-level application structure
 * - Initializes context-based state management
 *
 * Architecture:
 * - ApplicationProvider creates and injects VehicleStateManager
 * - VehicleDashboard and all children have access to context
 * - Clean separation: providers at root, pages/components below
 * - Structured for future routing, layout, and error handling
 *
 * Verifies: Requisito 5 (arquitectura limpia), Requisito 13.1
 *
 * @returns React component representing complete application
 *
 * @example
 * ```typescript
 * // In main.tsx
 * import { App } from './App.tsx';
 * import { createRoot } from 'react-dom/client';
 *
 * createRoot(document.getElementById('root')!).render(<App />);
 * ```
 */
export const App: React.FC = () => {
  return (
    <ApplicationProvider>
      <div className="app-root" data-testid="app-root">
        <VehicleDashboard />
      </div>
    </ApplicationProvider>
  );
};

export default App;

