import React from 'react';
import { ApplicationProvider as ContextProvider } from './ApplicationContext';
import { createDataProvider } from '@infrastructure/data-providers/data-provider-factory';
import { createRealtimePort } from '@infrastructure/sse/sse-connection-factory';
import { LocalStorageAdapter } from '@infrastructure/persistence/local-storage-adapter';

/**
 * ApplicationProvider - Wrapper component that initializes ApplicationContext with dependencies
 *
 * Responsibilities:
 * - Creates all infrastructure dependencies (DataProvider, SSE, Storage)
 * - Wraps children with ApplicationContext Provider
 * - Handles initialization of all required services
 * - Provides dependency injection for clean architecture
 *
 * Verifies: Requisito 5 (separación arquitectónica), Requisito 13.1
 *
 * @param props - Component props
 * @param props.children - React elements to wrap with provider
 * @returns React component providing ApplicationContext
 *
 * @example
 * ```typescript
 * // In App.tsx
 * export const App = () => (
 *   <ApplicationProvider>
 *     <VehicleDashboard />
 *   </ApplicationProvider>
 * );
 * ```
 */
interface ApplicationProviderProps {
  children: React.ReactNode;
}

export const ApplicationProvider: React.FC<ApplicationProviderProps> = ({ children }) => {
  // Create infrastructure components with environment-based configuration
  const dataProvider = createDataProvider();
  const realtimePort = createRealtimePort();
  const storage = new LocalStorageAdapter();

  return (
    <ContextProvider
      dataProvider={dataProvider}
      realtimePort={realtimePort}
      storage={storage}
    >
      {children}
    </ContextProvider>
  );
};

export default ApplicationProvider;
