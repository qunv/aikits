"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.configurePersistable = exports.mpsReactionOptions = exports.mpsConfig = void 0;
exports.mpsConfig = {};
exports.mpsReactionOptions = {};
const configurePersistable = (config, reactionOptions = {}) => {
    exports.mpsConfig = config;
    exports.mpsReactionOptions = reactionOptions;
};
exports.configurePersistable = configurePersistable;
