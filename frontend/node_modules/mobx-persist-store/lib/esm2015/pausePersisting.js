import { PersistStoreMap } from './PersistStoreMap';
export const pausePersisting = (target) => {
    var _a;
    (_a = PersistStoreMap.get(target)) === null || _a === void 0 ? void 0 : _a.pausePersisting();
};
