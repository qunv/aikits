import { PersistStoreMap } from './PersistStoreMap';
export const isHydrated = (target) => {
    var _a, _b;
    return (_b = (_a = PersistStoreMap.get(target)) === null || _a === void 0 ? void 0 : _a.isHydrated) !== null && _b !== void 0 ? _b : false;
};
