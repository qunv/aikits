import { makeAutoObservable } from 'mobx';
import { makePersistable } from 'mobx-persist-store';

export class UserStore {
  displayName: string = 'User';

  constructor() {
    makeAutoObservable(this);
    makePersistable(this, {
      name: 'UserStore',
      properties: ['displayName'],
      storage: typeof window !== 'undefined' ? window.localStorage : undefined,
    });
  }

  setDisplayName(name: string) {
    this.displayName = name;
  }
}
