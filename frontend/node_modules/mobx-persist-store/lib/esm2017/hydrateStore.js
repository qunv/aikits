"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.hydrateStore = void 0;
const PersistStoreMap_1 = require("./PersistStoreMap");
const hydrateStore = async (target) => {
    var _a;
    await ((_a = PersistStoreMap_1.PersistStoreMap.get(target)) === null || _a === void 0 ? void 0 : _a.hydrateStore());
};
exports.hydrateStore = hydrateStore;
