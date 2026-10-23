import React, { useEffect, useState } from 'react';
import { View, ActivityIndicator, Alert, SafeAreaView } from 'react-native';
import { CaptureScreen } from './presentation/screens/CaptureScreen';
import { DependencyContainer } from './infrastructure/di/DependencyContainer';
import { RealClock } from './infrastructure/platform/RealClock';
import { RealIDGenerator } from './infrastructure/platform/RealIDGenerator';

/**
 * App
 *
 * Root component that initializes the application.
 * Sets up dependency injection and mounts the main screen.
 */
export default function App() {
  const [container, setContainer] = useState<DependencyContainer | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const initializeApp = async () => {
      try {
        // Initialize real providers (would be replaced with actual implementations)
        const clock = new RealClock();
        const idGenerator = new RealIDGenerator();

        // TODO: Initialize real platform providers:
        // - RealLocationProvider (React Native location APIs)
        // - SQLitePersistenceProvider (sqlite3 driver)
        // - RealConnectivityProvider (NetInfo)
        // - RealApiClient (fetch with real backend URL)

        // For now, create container with mock providers for development
        // This will be replaced with real providers in implementation

        // Placeholder - actual implementation will use real providers
        console.log('[App] Initialized with real providers (clock, idGenerator)');
        console.log('[App] TODO: Wire up platform-specific providers');

        // Create container with stubs for now
        // In real implementation: new DependencyContainer(
        //   realLocationProvider,
        //   realPersistenceProvider,
        //   realConnectivityProvider,
        //   realApiClient,
        //   clock,
        //   idGenerator
        // )

        setError('Platform providers not yet implemented');
      } catch (err) {
        const msg = err instanceof Error ? err.message : 'Unknown initialization error';
        setError(msg);
        console.error('[App] Initialization failed:', err);
      }
    };

    initializeApp();
  }, []);

  if (error) {
    return (
      <SafeAreaView style={{ flex: 1, justifyContent: 'center', alignItems: 'center' }}>
        <View style={{ padding: 20 }}>
          <AlertError message={error} />
        </View>
      </SafeAreaView>
    );
  }

  if (!container) {
    return (
      <SafeAreaView style={{ flex: 1, justifyContent: 'center', alignItems: 'center' }}>
        <ActivityIndicator size="large" color="#0066cc" />
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={{ flex: 1 }}>
      <CaptureScreen container={container} />
    </SafeAreaView>
  );
}

const AlertError = ({ message }: { message: string }) => (
  <View style={{ backgroundColor: '#ffe6e6', padding: 16, borderRadius: 8 }}>
    <View style={{ fontSize: 14, color: '#cc0000', fontWeight: '600' }}>
      Error de inicialización
    </View>
  </View>
);
