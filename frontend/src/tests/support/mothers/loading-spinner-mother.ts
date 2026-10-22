/**
 * LoadingSpinnerMother - Object Mother para crear instancias de Test Data Builder de props del LoadingSpinner
 *
 * Proporciona escenarios predefinidos (default, withSmallSize, withStatusText, etc.)
 * y retorna valores que pueden ser utilizados directamente o modificados.
 *
 * Verifica: testing.md Sección 7 (Object Mother + Test Data Builder)
 */

/**
 * Props para el componente LoadingSpinner
 */
export interface LoadingSpinnerProps {
  size?: 'small' | 'medium' | 'large';
  statusText?: string;
  ariaLabel?: string;
}

/**
 * LoadingSpinnerBuilder - Fluent builder para construir props de LoadingSpinner
 */
export class LoadingSpinnerBuilder {
  private size: 'small' | 'medium' | 'large' = 'medium';
  private statusText?: string;
  private ariaLabel?: string;

  withSize(size: 'small' | 'medium' | 'large'): this {
    this.size = size;
    return this;
  }

  withStatusText(text: string): this {
    this.statusText = text;
    return this;
  }

  withAriaLabel(label: string): this {
    this.ariaLabel = label;
    return this;
  }

  build(): LoadingSpinnerProps {
    return {
      size: this.size,
      statusText: this.statusText,
      ariaLabel: this.ariaLabel,
    };
  }
}

/**
 * LoadingSpinnerMother - Point único de scenarios comunes
 */
export class LoadingSpinnerMother {
  /**
   * Retorna props válidas con valores por defecto (medium, sin texto)
   */
  static valid(): LoadingSpinnerBuilder {
    return new LoadingSpinnerBuilder();
  }

  /**
   * Retorna props para estado de conexión: "Conectando..."
   */
  static connecting(): LoadingSpinnerBuilder {
    return new LoadingSpinnerBuilder().withStatusText('Conectando...');
  }

  /**
   * Retorna props para estado de carga de vehículos: "Cargando vehículos..."
   */
  static loadingVehicles(): LoadingSpinnerBuilder {
    return new LoadingSpinnerBuilder().withStatusText('Cargando vehículos...');
  }

  /**
   * Retorna props para estado de sincronización: "Sincronizando datos..."
   */
  static synchronizing(): LoadingSpinnerBuilder {
    return new LoadingSpinnerBuilder().withStatusText('Sincronizando datos...');
  }

  /**
   * Retorna props con tamaño small
   */
  static small(): LoadingSpinnerBuilder {
    return new LoadingSpinnerBuilder().withSize('small');
  }

  /**
   * Retorna props con tamaño medium (por defecto)
   */
  static medium(): LoadingSpinnerBuilder {
    return new LoadingSpinnerBuilder().withSize('medium');
  }

  /**
   * Retorna props con tamaño large
   */
  static large(): LoadingSpinnerBuilder {
    return new LoadingSpinnerBuilder().withSize('large');
  }

  /**
   * Retorna props con tamaño small + status text
   */
  static smallWithText(): LoadingSpinnerBuilder {
    return new LoadingSpinnerBuilder()
      .withSize('small')
      .withStatusText('Cargando...');
  }

  /**
   * Retorna props con tamaño large + status text
   */
  static largeWithText(): LoadingSpinnerBuilder {
    return new LoadingSpinnerBuilder()
      .withSize('large')
      .withStatusText('Inicializando...');
  }

  /**
   * Retorna props con custom ARIA label
   */
  static withCustomAriaLabel(): LoadingSpinnerBuilder {
    return new LoadingSpinnerBuilder().withAriaLabel('Cargando contenido personalizado');
  }

  /**
   * Retorna props con largo status text para testing de truncación
   */
  static withLongStatusText(): LoadingSpinnerBuilder {
    return new LoadingSpinnerBuilder().withStatusText(
      'Este es un mensaje de estado muy largo que podría causar desbordamiento visual',
    );
  }
}

/**
 * Textos comunes para testing
 */
export const LoadingSpinnerStatusTexts = {
  CONNECTING: 'Conectando...',
  LOADING_VEHICLES: 'Cargando vehículos...',
  SYNCHRONIZING: 'Sincronizando datos...',
  INITIALIZING: 'Inicializando...',
  PROCESSING: 'Procesando solicitud...',
  SAVING: 'Guardando cambios...',
};

/**
 * Tamaños disponibles
 */
export const LoadingSpinnerSizes = {
  SMALL: 'small' as const,
  MEDIUM: 'medium' as const,
  LARGE: 'large' as const,
};

/**
 * Pixel sizes asociados a cada variant
 */
export const LoadingSpinnerPixelSizes = {
  small: 24,
  medium: 48,
  large: 64,
};
