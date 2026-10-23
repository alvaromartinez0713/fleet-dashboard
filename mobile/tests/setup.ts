// Setup file for Jest
// Mocks for React Native and Expo

jest.mock('expo-location', () => ({
  requestForegroundPermissionsAsync: jest.fn(async () => ({ status: 'granted' })),
  getCurrentPositionAsync: jest.fn(async () => ({
    coords: { latitude: 0, longitude: 0, accuracy: 10, heading: 0, speed: 0, altitude: 0 },
  })),
}));

jest.mock('expo-sqlite', () => ({
  openDatabaseAsync: jest.fn(async () => ({
    execAsync: jest.fn(async () => {}),
    runAsync: jest.fn(async () => {}),
    getAllAsync: jest.fn(async () => []),
    getFirstAsync: jest.fn(async () => null),
    closeAsync: jest.fn(async () => {}),
  })),
}));
