import { PersistStoreMap } from './PersistStoreMap';
export const hydrateStore = async (target) => {
    var _a;
    await ((_a = PersistStoreMap.get(target)) === null || _a === void 0 ? void 0 : _a.hydrateStore());
};
