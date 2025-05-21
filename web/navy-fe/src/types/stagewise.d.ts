declare module '@stagewise/toolbar-react' {
    export interface StagewiseConfig {
      plugins: any[];
    }
    
    export interface StagewiseToolbarProps {
      config: StagewiseConfig;
    }
    
    export const StagewiseToolbar: React.FC<StagewiseToolbarProps>;
  }
  
  declare module '@stagewise/toolbar' {
    export interface StagewiseConfig {
      plugins: any[];
    }
    
    export function initToolbar(config: StagewiseConfig): void;
  } 