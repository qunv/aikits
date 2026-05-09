"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.isPersisting = void 0;
const PersistStoreMap_1 = require("./PersistStoreMap");
const isPersisting = (target) => {
    var _a, _b;
    return (_b = (_a = PersistStoreMap_1.PersistStoreMap.get(target)) === null || _a === void 0 ? void 0 : _a.isPersisting) !== null && _b !== void 0 ? _b : false;
};
exports.isPersisting = isPersisting;
