import React, { createContext, useContext } from 'react';
import { PreferencesStore } from './subStores/PreferencesStore';
import { UserStore } from './subStores/UserStore';
import { RepoStore } from './subStores/RepoStore';

class RootStore {
  userStore: UserStore;
  preferencesStore: PreferencesStore;
  repoStore: RepoStore;

  constructor() {
    this.userStore = new UserStore();
    this.preferencesStore = new PreferencesStore();
    this.repoStore = new RepoStore();
  }
}

const rootStore = new RootStore();
const StoreContext = createContext<RootStore>(rootStore);

export function StoreProvider({ children }: { children: React.ReactNode }) {
  return <StoreContext.Provider value={rootStore}>{children}</StoreContext.Provider>;
}

export function useStore() {
  const context = useContext(StoreContext);
  if (!context) {
    throw new Error('useStore must be used within StoreProvider');
  }
  return context;
}

export function useUserStore() {
  const { userStore } = useStore();
  return userStore;
}

export function usePreferencesStore() {
  const { preferencesStore } = useStore();
  return preferencesStore;
}

export function useRepoStore() {
  const { repoStore } = useStore();
  return repoStore;
}

export { rootStore };
