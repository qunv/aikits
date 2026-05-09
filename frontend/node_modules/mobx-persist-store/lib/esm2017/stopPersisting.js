"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.stopPersisting = void 0;
const PersistStoreMap_1 = require("./PersistStoreMap");
const stopPersisting = (target) => {
    var _a;
    (_a = PersistStoreMap_1.PersistStoreMap.get(target)) === null || _a === void 0 ? void 0 : _a.stopPersisting();
};
exports.stopPersisting = stopPersisting;
