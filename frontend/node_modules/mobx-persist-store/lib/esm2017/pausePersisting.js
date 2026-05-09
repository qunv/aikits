"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.pausePersisting = void 0;
const PersistStoreMap_1 = require("./PersistStoreMap");
const pausePersisting = (target) => {
    var _a;
    (_a = PersistStoreMap_1.PersistStoreMap.get(target)) === null || _a === void 0 ? void 0 : _a.pausePersisting();
};
exports.pausePersisting = pausePersisting;
