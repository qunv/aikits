import { makeAutoObservable } from 'mobx';
import { makePersistable } from 'mobx-persist-store';

export class PreferencesStore {
  sidebarCollapsed: boolean = false;

  constructor() {
    makeAutoObservable(this);
    makePersistable(this, {
      name: 'PreferencesStore',
      properties: ['sidebarCollapsed'],
      storage: typeof window !== 'undefined' ? window.localStorage : undefined,
    });
  }

  setSidebarCollapsed(collapsed: boolean) {
    this.sidebarCollapsed = collapsed;
  }
}
