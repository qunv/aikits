export let mpsConfig = {};
export let mpsReactionOptions = {};
export const configurePersistable = (config, reactionOptions = {}) => {
    mpsConfig = config;
    mpsReactionOptions = reactionOptions;
};
