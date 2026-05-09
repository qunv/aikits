import { makeAutoObservable } from 'mobx';
import { makePersistable } from 'mobx-persist-store';

export class RepoStore {
  repoPath: string = '';

  constructor() {
    makeAutoObservable(this);
    makePersistable(this, {
      name: 'RepoStore',
      properties: ['repoPath'],
      storage: typeof window !== 'undefined' ? window.localStorage : undefined,
    });
  }

  setRepoPath(path: string) {
    this.repoPath = path;
  }

  get repoName(): string {
    if (!this.repoPath) return '';
    return this.repoPath.split('/').filter(Boolean).pop() ?? '';
  }
}
