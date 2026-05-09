export var mpsConfig = {};
export var mpsReactionOptions = {};
export var configurePersistable = function (config, reactionOptions) {
    if (reactionOptions === void 0) { reactionOptions = {}; }
    mpsConfig = config;
    mpsReactionOptions = reactionOptions;
};
