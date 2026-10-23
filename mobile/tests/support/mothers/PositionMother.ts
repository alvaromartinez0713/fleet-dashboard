import { Position } from '../../../src/domain/models/Position';

class PositionBuilder {
  private eventId = '00000000-0000-0000-0000-000000000001';
  private vehicleId = 'VH-001';
  private latitude = 40.7128;
  private longitude = -74.006;
  private capturedAt = new Date('2024-01-01T00:00:00Z');
  private status: 'pending' | 'sent' | 'error' | 'validation_error' = 'pending';

  withEventId(id: string): PositionBuilder {
    this.eventId = id;
    return this;
  }

  withVehicleId(id: string): PositionBuilder {
    this.vehicleId = id;
    return this;
  }

  withCoordinates(lat: number, lon: number): PositionBuilder {
    this.latitude = lat;
    this.longitude = lon;
    return this;
  }

  withStatus(s: 'pending' | 'sent' | 'error' | 'validation_error'): PositionBuilder {
    this.status = s;
    return this;
  }

  build(): Position {
    return new Position(
      this.eventId,
      this.vehicleId,
      this.latitude,
      this.longitude,
      this.capturedAt,
      this.status,
      new Date()
    );
  }
}

export const PositionMother = {
  valid: () => new PositionBuilder(),
  pending: () => new PositionBuilder().withStatus('pending'),
  sent: () => new PositionBuilder().withStatus('sent'),
  invalidCoordinates: () => new PositionBuilder().withCoordinates(120, 200),
};
