import { StorageOptions } from './types';
export declare class StorageAdapter {
    readonly options: StorageOptions;
    constructor(options: StorageOptions);
    setItem<T extends Record<string, unknown>>(key: string, item: T): Promise<void>;
    getItem<T extends Record<string, any>>(key: string): Promise<T>;
    removeItem(key: string): Promise<void>;
}
