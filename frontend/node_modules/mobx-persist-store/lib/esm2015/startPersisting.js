import { PersistStoreMap } from './PersistStoreMap';
export const startPersisting = (target) => {
    var _a;
    (_a = PersistStoreMap.get(target)) === null || _a === void 0 ? void 0 : _a.startPersisting();
};
