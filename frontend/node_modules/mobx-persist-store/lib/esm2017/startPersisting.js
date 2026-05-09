"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.startPersisting = void 0;
const PersistStoreMap_1 = require("./PersistStoreMap");
const startPersisting = (target) => {
    var _a;
    (_a = PersistStoreMap_1.PersistStoreMap.get(target)) === null || _a === void 0 ? void 0 : _a.startPersisting();
};
exports.startPersisting = startPersisting;
