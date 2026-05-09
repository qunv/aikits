"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.clearPersistedStore = void 0;
const PersistStoreMap_1 = require("./PersistStoreMap");
const clearPersistedStore = async (target) => {
    var _a;
    await ((_a = PersistStoreMap_1.PersistStoreMap.get(target)) === null || _a === void 0 ? void 0 : _a.clearPersistedStore());
};
exports.clearPersistedStore = clearPersistedStore;
